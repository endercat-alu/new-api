package relay

import (
	"bytes"
	"io"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/service"
	"github.com/QuantumNous/new-api/setting"
)

// maskedPassThroughBody 透传分支的脱敏兜底。
// 透传分支直接发送原始 body、不经过 ConvertXXXRequest，因此 controller 层对 info.Request
// 的结构化脱敏对它无效，需在此对原始字节做 body 级掩码。
// 未开启脱敏或未命中时，返回原始 storage 的只读 reader（零额外开销）。
func maskedPassThroughBody(storage common.BodyStorage) io.Reader {
	if setting.MaskSecretsEnabled {
		if raw, err := storage.Bytes(); err == nil {
			if masked, hit := service.MaskBytes(raw); hit {
				return bytes.NewReader(masked)
			}
		}
	}
	return common.ReaderOnly(storage)
}
