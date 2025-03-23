package server

import (
	"fmt"

	"google.golang.org/protobuf/reflect/protodesc"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/reflect/protoregistry"
	"google.golang.org/protobuf/types/descriptorpb"
	"google.golang.org/protobuf/types/dynamicpb"
)

// MessageFactory produces the Input/Output empty messages.
type MessageFactory func() (in protoreflect.Message, out protoreflect.Message)

// NewMessageFactory returns the constructor for the Input/Output protobuf objects
// by given proto descriptors.
func NewMessageFactory(
	protoDescr *descriptorpb.FileDescriptorProto,
	methodDescr *descriptorpb.MethodDescriptorProto,
) (MessageFactory, error) {
	fileDesc, err := protodesc.NewFile(protoDescr, protoregistry.GlobalFiles)
	if err != nil {
		return nil, fmt.Errorf("construct file descriptor: %w", err)
	}

	inputName := protoreflect.FullName(methodDescr.GetInputType())
	outputName := protoreflect.FullName(methodDescr.GetOutputType())

	inputMessageDesc := fileDesc.Messages().ByName(inputName.Name())
	outputMessageDesc := fileDesc.Messages().ByName(outputName.Name())

	var inputMsgType protoreflect.MessageType
	var outMsg protoreflect.Message
	if inputMessageDesc != nil {
		inputMsgType = dynamicpb.NewMessageType(inputMessageDesc)
	}
	if outputMessageDesc != nil {
		outMsg = dynamicpb.NewMessage(outputMessageDesc)
	}

	return func() (in protoreflect.Message, out protoreflect.Message) {
		return inputMsgType.New(), outMsg
	}, nil
}
