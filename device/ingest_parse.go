package device

import (
	"encoding/base64"
	"encoding/json"
	"mime"
	"time"
)

// uplinkMessage is an uplink as a network server sends it, in the fields
// the platform keeps (device:N9).
type uplinkMessage struct {
	DevEUI  string
	FCnt    int64
	Payload []byte
	// ReceivedAt is the time the network server received the uplink, or nil
	// when the body does not say.
	ReceivedAt *time.Time
}

// chirpStackUp holds the fields of a ChirpStack v4 HTTP integration "up"
// event in JSON encoding (device:F1). Other fields are ignored.
type chirpStackUp struct {
	Time       *time.Time `json:"time"`
	DeviceInfo struct {
		DevEUI string `json:"devEui"`
	} `json:"deviceInfo"`
	FCnt int64  `json:"fCnt"`
	Data string `json:"data"`
}

// parseChirpStack reads a ChirpStack request. An event that is not "up" is
// skipped. A ChirpStack integration set to Protobuf sends a binary body,
// which the platform does not read.
func parseChirpStack(event, contentType string, body []byte) (msg *uplinkMessage, skip bool, err *ruleError) {
	if event != "up" {
		return nil, true, nil
	}
	mediaType, _, _ := mime.ParseMediaType(contentType)
	if mediaType == "application/octet-stream" || mediaType == "application/x-protobuf" {
		return nil, false, errChirpStackProtobuf
	}
	var up chirpStackUp
	if err := json.Unmarshal(body, &up); err != nil {
		return nil, false, errIngestInvalidJSON
	}
	msg, err = newUplinkMessage(up.DeviceInfo.DevEUI, up.FCnt, up.Data, up.Time)
	return msg, false, err
}

// ttnWebhook holds the fields of a The Things Stack v3 webhook message
// (device:F2). Only a message with uplink_message is an uplink.
type ttnWebhook struct {
	ReceivedAt   *time.Time `json:"received_at"`
	EndDeviceIDs struct {
		DevEUI string `json:"dev_eui"`
	} `json:"end_device_ids"`
	UplinkMessage *struct {
		FCnt       int64      `json:"f_cnt"`
		FRMPayload string     `json:"frm_payload"`
		ReceivedAt *time.Time `json:"received_at"`
	} `json:"uplink_message"`
}

// parseTTN reads a TTN webhook request. A message that is not an uplink,
// such as a join accept, is skipped.
func parseTTN(body []byte) (msg *uplinkMessage, skip bool, err *ruleError) {
	var hook ttnWebhook
	if err := json.Unmarshal(body, &hook); err != nil {
		return nil, false, errIngestInvalidJSON
	}
	up := hook.UplinkMessage
	if up == nil {
		return nil, true, nil
	}
	at := up.ReceivedAt
	if at == nil {
		at = hook.ReceivedAt
	}
	msg, err = newUplinkMessage(hook.EndDeviceIDs.DevEUI, up.FCnt, up.FRMPayload, at)
	return msg, false, err
}

// newUplinkMessage checks the DevEUI and decodes the base64 payload. A
// missing frame counter is 0: both network servers leave out a zero value.
func newUplinkMessage(devEUI string, fCnt int64, data string, at *time.Time) (*uplinkMessage, *ruleError) {
	eui, err := normalizeDevEUI(devEUI)
	if err != nil {
		return nil, errIngestDevEUI
	}
	payload, err := base64.StdEncoding.DecodeString(data)
	if err != nil {
		return nil, errIngestPayload
	}
	return &uplinkMessage{DevEUI: eui, FCnt: fCnt, Payload: payload, ReceivedAt: at}, nil
}
