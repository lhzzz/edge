package protoerr

import (
	"edge/api/edge-proto/pb"
)

func ParamErr(validateErr string) *pb.Error {
	return &pb.Error{
		Code: pb.ErrorCode_PARAMETER_FAILED,
		Msg:  validateErr,
	}
}

func InternalErr(err error) *pb.Error {
	if err != nil {
		return &pb.Error{
			Code: pb.ErrorCode_INTERNAL_ERROR,
			Msg:  err.Error(),
		}
	}
	return nil
}

func StreamFinishErr(msg string) *pb.Error {
	return &pb.Error{
		Code: pb.ErrorCode_SERVICE_STREAM_CALL_FINISH,
		Msg:  msg,
	}
}

func NotFoundErr(msg string) *pb.Error {
	return &pb.Error{
		Code: pb.ErrorCode_NO_RESULT,
		Msg:  msg,
	}
}

func IsNotFoundErr(perr *pb.Error) bool {
	return perr.Code == pb.ErrorCode_NO_RESULT
}
