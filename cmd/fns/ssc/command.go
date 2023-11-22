/*
 * Copyright 2023 Wang Min Xiang
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
 *
 */

package ssc

import (
	"fmt"
	"github.com/aacfactory/afssl"
	"github.com/aacfactory/errors"
	"github.com/urfave/cli/v2"
	"os"
	"path/filepath"
	"strings"
)

var Command = &cli.Command{
	Name:        "ssc",
	Aliases:     nil,
	Usage:       "fns scc --sa=ECDSA --cn={CN} {output dir}",
	Description: "create self sign cert",
	ArgsUsage:   "",
	Category:    "",
	Flags: []cli.Flag{
		&cli.StringFlag{
			Name:     "cn",
			Required: true,
			Usage:    "CN",
		},
		&cli.StringFlag{
			Name:     "sa",
			Required: true,
			Usage:    "signature algorithm",
		},
		&cli.IntFlag{
			Name:     "expire",
			Required: true,
			Usage:    "expire days",
		},
	},
	Action: func(ctx *cli.Context) (err error) {
		// dst
		dstDir := strings.TrimSpace(ctx.Args().First())
		if dstDir == "" {
			dstDir = "."
		}
		if !filepath.IsAbs(dstDir) {
			dstDir, err = filepath.Abs(dstDir)
			if err != nil {
				err = errors.Warning("fns: create self sign cert").WithCause(err).WithMeta("dir", dstDir)
				return
			}
		}
		dstDir = filepath.ToSlash(dstDir)
		// cn
		cn := ctx.String("cn")
		// sa
		var keyType afssl.KeyType
		sa := strings.ToUpper(ctx.String("sa"))
		switch sa {
		case "ECDSA", "":
			keyType = afssl.ECDSA()
			break
		case "RSA":
			keyType = afssl.RSA()
			break
		case "ED25519":
			keyType = afssl.ED25519()
			break
		case "SM2":
			keyType = afssl.SM2()
			break
		default:
			err = errors.Warning("fns: create self sign cert").WithCause(fmt.Errorf("sa is not supported")).WithMeta("sa", sa)
			return
		}
		// expire
		expire := ctx.Int("expire")
		if expire < 1 {
			expire = 365
		}

		config := afssl.CertificateConfig{
			Subject: &afssl.CertificatePkixName{
				CommonName: cn,
			},
			IPs:      nil,
			Emails:   nil,
			DNSNames: nil,
		}
		cert, key, genErr := afssl.GenerateCertificate(config, afssl.CA(), afssl.WithKeyType(keyType), afssl.WithExpirationDays(expire))
		if genErr != nil {
			err = errors.Warning("fns: create self sign cert").WithCause(genErr)
			return
		}

		err = os.WriteFile(filepath.Join(dstDir, "ca.crt"), cert, 0644)
		if err != nil {
			err = errors.Warning("fns: create self sign cert").WithCause(err)
			return
		}
		err = os.WriteFile(filepath.Join(dstDir, "ca.key"), key, 0644)
		if err != nil {
			err = errors.Warning("fns: create self sign cert").WithCause(err)
			return
		}
		fmt.Println("fns: create self sign cert created!")
		return
	},
}
