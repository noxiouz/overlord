package overlord

import (
	"github.com/cocaine/cocaine-framework-go/vendor/src/github.com/ugorji/go/codec"
)

var (
	mhAsocket = codec.MsgpackHandle{
		BasicHandle: codec.BasicHandle{
			EncodeOptions: codec.EncodeOptions{
				StructToArray: true,
			},
		},
	}
	hAsocket = &mhAsocket

	mPayloadHandler codec.MsgpackHandle
	payloadHandler  = &mPayloadHandler
)

func convertPayload(in interface{}, out interface{}) error {
	var buf []byte
	if err := codec.NewEncoderBytes(&buf, payloadHandler).Encode(in); err != nil {
		return err
	}
	if err := codec.NewDecoderBytes(buf, payloadHandler).Decode(out); err != nil {
		return err
	}
	return nil
}
