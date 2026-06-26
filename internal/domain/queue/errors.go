package queue

import "errors"

var (
	ErrOpeningChannel        = errors.New("Channel setup error")
	ErrDeclareQueue          = errors.New("Declare queue error")
	ErrMessageEncodingFailed = errors.New("Json encoding failed")
	ErrPublishMessageFailed  = errors.New("Publish message failed")
	ErrQoSFailed             = errors.New("Quality Of Service failed")
	ErrConsumerChannelClosed = errors.New("Consumer channel closed")
)
