package services

import "github.com/aacfactory/fns/commons/versions"

type Header struct {
	processId        []byte
	requestId        []byte
	deviceId         []byte
	deviceIp         []byte
	authorization    []byte
	acceptedVersions versions.Intervals
	internal         bool
}

func (header Header) ProcessId() []byte {
	return header.processId
}

func (header Header) RequestId() []byte {
	return header.requestId
}

func (header Header) DeviceId() []byte {
	return header.deviceId
}

func (header Header) DeviceIp() []byte {
	return header.deviceIp
}

func (header Header) Authorization() []byte {
	return header.authorization
}

func (header Header) AcceptedVersions() versions.Intervals {
	if len(header.acceptedVersions) == 0 {
		return versions.AllowAllIntervals()
	}
	return header.acceptedVersions
}

func (header Header) Internal() bool {
	return header.internal
}