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

package signatures_test

import (
	"fmt"
	"github.com/aacfactory/fns/commons/signatures"
	"testing"
	"time"
)

func TestECC(t *testing.T) {
	pub := []byte(`MFkwEwYHKoZIzj0CAQYIKoZIzj0DAQcDQgAEgTozHvlIiW85B63PRWb/GycHQ2pw8RXKQZErDqBFQr26Hy7Xtjd/c/bS4zm7+Q8twrnH8bJcNGrJKbmn+wEOhA==`)
	pri := []byte(`MHcCAQEEIOcApuQLLvGy0fBpPH5rUNgj3qq0+0J86nB8ULcqfpd9oAoGCCqGSM49AwEHoUQDQgAEgTozHvlIiW85B63PRWb/GycHQ2pw8RXKQZErDqBFQr26Hy7Xtjd/c/bS4zm7+Q8twrnH8bJcNGrJKbmn+wEOhA==`)
	s, sErr := signatures.ECC(pub, pri, signatures.XXHash)
	if sErr != nil {
		t.Errorf("%+v", sErr)
		return
	}
	p := []byte(time.Now().String())
	v := s.Sign(p)
	fmt.Println(s.Verify(p, v))
	fmt.Println(len(v))
}
