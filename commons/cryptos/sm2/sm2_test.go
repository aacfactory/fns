/*
 * Copyright 2021 Wang Min Xiang
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 * 	http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package sm2_test

import (
	"bytes"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"github.com/aacfactory/fns/commons/cryptos/sm2"
	"testing"
	"time"
)

func TestSM2(t *testing.T) {
	ppu := []byte(`MFkwEwYHKoZIzj0CAQYIKoEcz1UBgi0DQgAEBSBsxc9x0b4INJ56Slqd0//Uj61tdntnA7A2mbXJlK7067MODxloEszWBdxAG48i0BOM1FglrosgIyTlLMyD8A==`)
	ppr := []byte(`MIH8MFcGCSqGSIb3DQEFDTBKMCkGCSqGSIb3DQEFDDAcBAjZnhKmWZifAwICCAAwDAYIKoZIhvcNAgcFADAdBglghkgBZQMEASoEEGJOmZt+D+ubFEwGj9UHZ+IEgaCBXuGuckH5/POTEc5oeejuX5nMz8+rrqZv3s0CgRsavD8mUJYBEWfxS8u5D7EwMdU77KeOfbGY7AhIjmLWW+5AGS3TAWZJOIEHSQ/VYki0HFXtFpr7rk+NWOYbZBtEJK5Ec6iFNjS27LtDJ25zzmrfz4GifwHjtsCBt9RXzzPzDAV+Gbb73CcYS+dnciTelxQcxB7MwImDWLM2aTkWSAuv`)
	pub, pubErr := sm2.ParsePublicKey(ppu)
	if pubErr != nil {
		t.Error(pubErr)
		return
	}
	pri, priErr := sm2.ParsePrivateKeyWithPassword(ppr, []byte("123456"))
	if priErr != nil {
		t.Error(priErr)
		return
	}
	p := []byte(time.Now().String())
	v, _ := pri.Sign(rand.Reader, p)
	fmt.Println(len(v))
	fmt.Println(pub.Verify(p, v))

	pubp, pubpErr := pub.Encode()
	if pubpErr != nil {
		t.Error(pubpErr)
		return
	}
	fmt.Println(bytes.Equal(ppu, pubp))

	prip, pripErr := pri.Encode()
	if pripErr != nil {
		t.Error(pripErr)
		return
	}
	pri, priErr = sm2.ParsePrivateKeyWithPassword(prip, []byte("123456"))
	p = []byte(time.Now().String())
	v, _ = pri.Sign(rand.Reader, p)
	fmt.Println(len(v), len(base64.URLEncoding.EncodeToString(v)))
	fmt.Println(pub.Verify(p, v))
}
