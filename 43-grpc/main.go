// Tema 43: gRPC y Protocol Buffers
// gRPC es un framework RPC (llamadas a procedimientos remotos) sobre HTTP/2.
// El contrato se define en un archivo .proto (proto/catalogo.proto) y a partir
// de él se GENERA código para servidor y cliente en muchos lenguajes.
//   - Protocol Buffers: formato binario compacto y tipado (más chico y rápido
//     que JSON), con compatibilidad hacia adelante/atrás por número de campo.
//   - Soporta streaming en ambas direcciones, deadlines, metadata (headers)
//     y códigos de error estándar.
//
// Estructura:
//
//	proto/catalogo.proto          contrato (lo escribimos a mano)
//	catalogopb/*.pb.go            código GENERADO (no editar)
//	main.go                       implementación del servidor + cliente
//
// El código generado ya está incluido, así que no necesitas protoc para
// ejecutar el ejemplo. Para regenerarlo tras cambiar el .proto:
//
//	go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
//	go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest
//	protoc --go_out=43-grpc/catalogopb --go_opt=paths=source_relative \
//	       --go-grpc_out=43-grpc/catalogopb --go-grpc_opt=paths=source_relative \
//	       -I 43-grpc/proto 43-grpc/proto/catalogo.proto
//
// (o con buf: https://buf.build, que no requiere instalar protoc)
//
// Ejecutar con: go run ./43-grpc
package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"sync"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"

	pb "golang-basico/43-grpc/catalogopb"
)

// ======================= SERVIDOR =======================

// servidorCatalogo implementa la interfaz pb.CatalogoServer generada.
// Embeber UnimplementedCatalogoServer es obligatorio: si en el futuro se
// agrega un método al .proto, el servidor sigue compilando (y ese método
// responde codes.Unimplemented).
type servidorCatalogo struct {
	pb.UnimplementedCatalogoServer

	mu        sync.Mutex
	productos map[int32]*pb.Producto
	siguiente int32
}

func nuevoServidor() *servidorCatalogo {
	s := &servidorCatalogo{productos: map[int32]*pb.Producto{}, siguiente: 1}
	for _, p := range []*pb.Producto{
		{Nombre: "El Aleph", Precio: 11, Categoria: pb.Categoria_CATEGORIA_LIBROS, Etiquetas: []string{"cuentos"}},
		{Nombre: "Teclado", Precio: 45.5, Categoria: pb.Categoria_CATEGORIA_ELECTRONICA},
		{Nombre: "Rayuela", Precio: 15, Categoria: pb.Categoria_CATEGORIA_LIBROS},
	} {
		p.Id = s.siguiente
		s.productos[p.Id] = p
		s.siguiente++
	}
	return s
}

// Unario. Los errores se devuelven con status.Error y un código gRPC
// (NotFound, InvalidArgument...), el equivalente a los status HTTP.
func (s *servidorCatalogo) ObtenerProducto(ctx context.Context, req *pb.ObtenerProductoRequest) (*pb.Producto, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	p, ok := s.productos[req.GetId()] // los Get* son seguros aunque req sea nil
	if !ok {
		return nil, status.Errorf(codes.NotFound, "producto %d no existe", req.GetId())
	}
	return p, nil
}

func (s *servidorCatalogo) CrearProducto(ctx context.Context, req *pb.CrearProductoRequest) (*pb.Producto, error) {
	// Leer metadata (los "headers" de gRPC) enviada por el cliente
	md, _ := metadata.FromIncomingContext(ctx)
	if roles := md.Get("rol"); len(roles) == 0 || roles[0] != "admin" {
		return nil, status.Error(codes.PermissionDenied, "solo un admin puede crear productos")
	}
	if req.GetNombre() == "" || req.GetPrecio() <= 0 {
		return nil, status.Error(codes.InvalidArgument, "nombre y precio > 0 son obligatorios")
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	p := &pb.Producto{
		Id:        s.siguiente,
		Nombre:    req.GetNombre(),
		Precio:    req.GetPrecio(),
		Etiquetas: req.GetEtiquetas(),
		Categoria: req.GetCategoria(),
	}
	s.productos[p.Id] = p
	s.siguiente++
	return p, nil
}

// Streaming del servidor: se llama stream.Send varias veces y se termina
// retornando nil (o un error)
func (s *servidorCatalogo) ListarProductos(req *pb.ListarProductosRequest, stream grpc.ServerStreamingServer[pb.Producto]) error {
	s.mu.Lock()
	var lista []*pb.Producto
	for id := int32(1); id < s.siguiente; id++ {
		if p, ok := s.productos[id]; ok && (req.GetPrecioMaximo() == 0 || p.Precio <= req.GetPrecioMaximo()) {
			lista = append(lista, p)
		}
	}
	s.mu.Unlock()

	for _, p := range lista {
		// Respetar la cancelación del cliente
		if err := stream.Context().Err(); err != nil {
			return status.FromContextError(err).Err()
		}
		if err := stream.Send(p); err != nil {
			return err
		}
		time.Sleep(20 * time.Millisecond) // simular trabajo entre envíos
	}
	return nil
}

// Streaming del cliente: se lee con Recv hasta io.EOF y se responde una vez
func (s *servidorCatalogo) Comprar(stream grpc.ClientStreamingServer[pb.ItemCompra, pb.ResumenCompra]) error {
	resumen := &pb.ResumenCompra{}
	for {
		item, err := stream.Recv()
		if errors.Is(err, io.EOF) {
			return stream.SendAndClose(resumen) // el cliente terminó de enviar
		}
		if err != nil {
			return err
		}
		s.mu.Lock()
		p, ok := s.productos[item.GetProductoId()]
		s.mu.Unlock()
		if !ok {
			return status.Errorf(codes.NotFound, "producto %d no existe", item.GetProductoId())
		}
		resumen.Cantidad += item.GetUnidades()
		resumen.Total += float64(item.GetUnidades()) * p.GetPrecio()
	}
}

// Interceptor unario: el "middleware" de gRPC (comparar con el tema 34)
func interceptorLog(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
	inicio := time.Now()
	resp, err := handler(ctx, req)
	fmt.Printf("   [servidor] %s -> %s (%v)\n", info.FullMethod, status.Code(err), time.Since(inicio).Round(time.Microsecond))
	return resp, err
}

// ======================= CLIENTE =======================

func main() {
	// Escuchar en un puerto libre elegido por el sistema (":0")
	lis, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		log.Fatal(err)
	}
	srv := grpc.NewServer(grpc.UnaryInterceptor(interceptorLog))
	pb.RegisterCatalogoServer(srv, nuevoServidor())
	go func() {
		// Serve bloquea hasta que el servidor se detiene
		if err := srv.Serve(lis); err != nil {
			log.Println("servidor gRPC detenido:", err)
		}
	}()
	defer srv.GracefulStop() // termina los RPC en curso antes de cerrar
	fmt.Println("Servidor gRPC en", lis.Addr())

	// Conexión del cliente. insecure = sin TLS (solo para desarrollo local;
	// en producción se usan credenciales TLS, ver tema 44).
	conn, err := grpc.NewClient(lis.Addr().String(), grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatal(err)
	}
	defer conn.Close()
	cliente := pb.NewCatalogoClient(conn) // cliente generado, tipado

	// Todo llamado lleva un context: siempre conviene ponerle deadline
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	fmt.Println("\n1) Llamada unaria")
	p, err := cliente.ObtenerProducto(ctx, &pb.ObtenerProductoRequest{Id: 1})
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("   recibido: id=%d nombre=%q precio=%.2f categoria=%s\n", p.Id, p.Nombre, p.Precio, p.Categoria)

	fmt.Println("\n2) Errores con códigos gRPC")
	_, err = cliente.ObtenerProducto(ctx, &pb.ObtenerProductoRequest{Id: 99})
	st, _ := status.FromError(err) // extraer código y mensaje del error
	fmt.Printf("   código=%s mensaje=%q\n", st.Code(), st.Message())

	fmt.Println("\n3) Metadata (headers) y validación")
	nuevo := &pb.CrearProductoRequest{Nombre: "Mouse", Precio: 19.9, Categoria: pb.Categoria_CATEGORIA_ELECTRONICA}
	_, err = cliente.CrearProducto(ctx, nuevo)
	fmt.Println("   sin rol:", status.Code(err))
	ctxAdmin := metadata.AppendToOutgoingContext(ctx, "rol", "admin")
	creado, err := cliente.CrearProducto(ctxAdmin, nuevo)
	fmt.Printf("   con rol admin: creado id=%d, err=%v\n", creado.GetId(), err)
	_, err = cliente.CrearProducto(ctxAdmin, &pb.CrearProductoRequest{Nombre: "Gratis"})
	fmt.Println("   precio 0:", status.Code(err))

	fmt.Println("\n4) Streaming del servidor (productos con precio <= 20)")
	stream, err := cliente.ListarProductos(ctx, &pb.ListarProductosRequest{PrecioMaximo: 20})
	if err != nil {
		log.Fatal(err)
	}
	for {
		p, err := stream.Recv()
		if errors.Is(err, io.EOF) {
			break // el servidor terminó el stream
		}
		if err != nil {
			log.Fatal(err)
		}
		fmt.Printf("   <- %s (%.2f)\n", p.Nombre, p.Precio)
	}

	fmt.Println("\n5) Streaming del cliente (carrito de compras)")
	compra, err := cliente.Comprar(ctx)
	if err != nil {
		log.Fatal(err)
	}
	for _, item := range []*pb.ItemCompra{{ProductoId: 1, Unidades: 2}, {ProductoId: 3, Unidades: 1}} {
		fmt.Printf("   -> producto %d x%d\n", item.ProductoId, item.Unidades)
		if err := compra.Send(item); err != nil {
			log.Fatal(err)
		}
	}
	resumen, err := compra.CloseAndRecv()
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("   resumen: %d unidades, total %.2f\n", resumen.Cantidad, resumen.Total)

	fmt.Println("\n6) Deadline excedido")
	ctxCorto, cancelCorto := context.WithTimeout(context.Background(), 30*time.Millisecond)
	defer cancelCorto()
	stream, _ = cliente.ListarProductos(ctxCorto, &pb.ListarProductosRequest{})
	var recibidos int
	for {
		if _, err = stream.Recv(); err != nil {
			break
		}
		recibidos++
	}
	fmt.Printf("   recibidos %d antes de cortar, código=%s\n", recibidos, status.Code(err))

	// 7) Protocol Buffers como formato de serialización, sin gRPC
	fmt.Println("\n7) Protobuf binario vs JSON")
	binario, _ := proto.Marshal(p)
	comoJSON, _ := protojson.Marshal(p)
	fmt.Printf("   binario: %d bytes %x\n", len(binario), binario)
	fmt.Printf("   JSON:    %d bytes %s\n", len(comoJSON), comoJSON)
	var decodificado pb.Producto
	if err := proto.Unmarshal(binario, &decodificado); err != nil {
		log.Fatal(err)
	}
	fmt.Println("   ¿iguales tras decodificar?", proto.Equal(p, &decodificado))
}
