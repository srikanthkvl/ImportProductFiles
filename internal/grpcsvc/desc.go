package grpcsvc

import (
	"context"

	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/structpb"
)

// ImporterServer defines the gRPC server interface.
type ImporterServer interface {
	Enqueue(context.Context, *structpb.Struct) (*structpb.Struct, error)
}

// ImporterServiceDesc describes the Importer service for manual registration.
var ImporterServiceDesc = grpc.ServiceDesc{
	ServiceName: "importer.Importer",
	HandlerType: (*ImporterServer)(nil),
	Methods: []grpc.MethodDesc{
		{
			MethodName: "Enqueue",
			Handler:    _Importer_Enqueue_Handler,
		},
	},
	Streams:  []grpc.StreamDesc{},
	Metadata: "importer",
}

func _Importer_Enqueue_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(structpb.Struct)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(ImporterServer).Enqueue(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/importer.Importer/Enqueue",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(ImporterServer).Enqueue(ctx, req.(*structpb.Struct))
	}
	return interceptor(ctx, in, info, handler)
}


