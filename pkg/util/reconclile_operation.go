package util

import "time"

type OperationResult struct {
	RequeueDelay   time.Duration
	RequeueRequest bool
	CancelRequest  bool
}

func (r OperationResult) RequeueOrCancel() bool {
	return r.RequeueRequest || r.CancelRequest
}

func ContinueOperationResult() OperationResult {
	return OperationResult{
		RequeueDelay:   0,
		RequeueRequest: false,
		CancelRequest:  false,
	}
}
func StopOperationResult() OperationResult {
	return OperationResult{
		RequeueDelay:   0,
		RequeueRequest: false,
		CancelRequest:  true,
	}
}

func StopProcessing() (result OperationResult, err error) {
	result = StopOperationResult()
	return
}

func Requeue() (result OperationResult, err error) {
	result = OperationResult{
		RequeueDelay:   0,
		RequeueRequest: true,
		CancelRequest:  false,
	}
	return
}

func RequeueWithError(errIn error) (result OperationResult, err error) {
	result = OperationResult{
		RequeueDelay:   0,
		RequeueRequest: true,
		CancelRequest:  false,
	}
	err = errIn
	return
}

func RequeueOnErrorOrStop(errIn error) (result OperationResult, err error) {
	result = OperationResult{
		RequeueDelay:   0,
		RequeueRequest: false,
		CancelRequest:  true,
	}
	err = errIn
	return
}

func RequeueOnErrorOrContinue(errIn error) (result OperationResult, err error) {
	result = OperationResult{
		RequeueDelay:   0,
		RequeueRequest: false,
		CancelRequest:  false,
	}
	err = errIn
	return
}

func RequeueAfter(delay time.Duration, errIn error) (result OperationResult, err error) {
	result = OperationResult{
		RequeueDelay:   delay,
		RequeueRequest: true,
		CancelRequest:  false,
	}
	err = errIn
	return
}

func ContinueProcessing() (result OperationResult, err error) {
	result = ContinueOperationResult()
	return
}
