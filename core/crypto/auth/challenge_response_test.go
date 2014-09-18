/*
Copyright (c) 2014
  Dario Brandes
  Thies Johannsen
  Paul Kröger
  Sergej Mann
  Roman Naumann
  Sebastian Thobe
All rights reserved.

Redistribution and use in source and binary forms, with or without modification,
are permitted provided that the following conditions are met:

1. Redistributions of source code must retain the above copyright notice, this
   list of conditions and the following disclaimer.

2. Redistributions in binary form must reproduce the above copyright notice,
   this list of conditions and the following disclaimer in the documentation
   and/or other materials provided with the distribution.

THIS SOFTWARE IS PROVIDED BY THE COPYRIGHT HOLDERS AND CONTRIBUTORS "AS IS" AND
ANY EXPRESS OR IMPLIED WARRANTIES, INCLUDING, BUT NOT LIMITED TO, THE IMPLIED
WARRANTIES OF MERCHANTABILITY AND FITNESS FOR A PARTICULAR PURPOSE ARE
DISCLAIMED. IN NO EVENT SHALL THE COPYRIGHT HOLDER OR CONTRIBUTORS BE LIABLE FOR
ANY DIRECT, INDIRECT, INCIDENTAL, SPECIAL, EXEMPLARY, OR CONSEQUENTIAL DAMAGES
(INCLUDING, BUT NOT LIMITED TO, PROCUREMENT OF SUBSTITUTE GOODS OR SERVICES;
LOSS OF USE, DATA, OR PROFITS; OR BUSINESS INTERRUPTION) HOWEVER CAUSED AND ON
ANY THEORY OF LIABILITY, WHETHER IN CONTRACT, STRICT LIABILITY, OR TORT
(INCLUDING NEGLIGENCE OR OTHERWISE) ARISING IN ANY WAY OUT OF THE USE OF THIS
SOFTWARE, EVEN IF ADVISED OF THE POSSIBILITY OF SUCH DAMAGE.
*/
/* -*- Mode: Go; indent-tabs-mode: t; c-basic-offset: 4; tab-width: 4 -*- */

package auth

import "testing"
import "crypto/sha256"
import _ "fmt"
import "../../crypto"

func TestChellengeResponse(t *testing.T) {
	A_PrivKeyPem := []byte(`-----BEGIN RSA PRIVATE KEY-----
MIICXAIBAAKBgQCxO9MKOEVPy/Y3+q6W33pQS4DwrO4k0zXaysnuliaREYASvoyF
P1l0ohffdyTuEUPz4MMLgoAtQeakwT2JIhPUVUae9BPzRFOl7ZSPVfNEWZxPnGMv
uePIklI0TACyPSByRZ+j7YvmKSOOXfyU0TiyB2PDxmQEPVxhxyy0w/GE0QIDAQAB
AoGAGAyzJXbfSOW3Yn88w7JNianFNGNy6UJT032jCyIK17KVO3Xp4YboH6CDNsqX
E0r6epRsQxqRRBLmNkMWk44xPGvgETyK3l23xNK6plKk6B6xONFo7b2D40C/Ids/
vw/IPtADgnJ+EXcN3pwbqA1UobKsh6K0QYmLhUzvqNH0xOUCQQDl2cHf/gkJiGfV
U2nP71QY1JMMNuMjYgQ2vVdXrg1vF2TVCQ+FgocnsC+Hn/9Qjd9AnlRmCte1k8k4
q7nQh+QTAkEAxWWiiZ5d0fLsf7qf+T3r8lOgC9G0AXXMZlYtWDlcMIjZJ90fY3HW
11N8BxcAgMqhvnIAmTSFhAolzL3mYSFoCwJAKR1ho7qiTTU8NZmdJNfPuD1WLGop
CSxASrZlyEZXGtcZb04Wm7A0kvHeHqmNFxC5rapkuHgaC93qsZtuOpIERwJBAL1Y
kGCRmE0bR9/9lBXwX7NCo/KyZIhCBp7javuFifjETAkBAmrRd9N0MTRbzA++Twfv
MsPCrY/KbBfI0IO7F6ECQFXdGVtPjYC+99JFbSoWlRj8nripWTmvlp3aYPD3ZAJU
RAczb2wbylOLxb3pidW9Tweo+kRXYYuXNFZ3vXnAjkE=
-----END RSA PRIVATE KEY-----`)

	B_PrivKeyPem := []byte(`-----BEGIN RSA PRIVATE KEY-----
MIICXwIBAAKBgQC6vjPYmMdzhnwLpmRvrUl6VhWDl+gAWOhmOFuVV9c8+EkQIqQH
g6E/0Y/+Tbj+aIiqUFxQ8vLdzXSrwR1XrkF8sUtF43QJaAGhMn5yPKawop4f/q+6
sBzomOHz+8t9yDJbC1NEJ6AcogGxPzga1CFCOptFjSzaAfay3I1G2nHMYwIDAQAB
AoGBAKraCiVI4nJXwHYILivepzM+P0C+YoyuyzF6ro/cZhhqMK6Kgvg8/fKdkNhh
07cvfJoWG+AT5w/3QZ9Cd84Yp+AUR4rJzSBhCHmmoTnZ1FGZWxNmfdoNn6pBvoz6
1wZtOd6pszLVce1YShv2AHLZr+rhaRe7BCcJDiEEGcX4sz4BAkEA62rjA9rPsElX
9fM5O9/aI2RdK5hqvgDFK9ntzb3BJ886l3Fi0/ifJjdiKI1nALRC54vYW4/PsECK
F+e2u6U9IQJBAMsR4eTtYHVik7NcUwRWNeensHEgQkS55+uCCQJIV+HLfcB8jcc3
2yoOml49On6JOZbPGxCi30te4a4GAs5SdQMCQQCyIkEBC7MX24eZbZ+jNLFlEm6F
rGEowIBxvAd7JNhhfScCrSNw7bHPQx0dPlHwcHYnquPd9KXc4hkcGZNlzZTBAkEA
jc4mBdwx4Kb+52BQZJXjPKqgDs9tF1sO9imvKtXj8LxOS01vIDAELuFVsPtmzpf7
DDIB/2MNNS/DvudZrERuiwJBAJ0PHZfkEYXaJb1bjYzHuc6o0gtggVcH2B6JrqRs
W90vDdM0H6z3hJdyrhPWcQxlkvrwVoXTsFvEQzZU2O7xSmo=
-----END RSA PRIVATE KEY-----`)

	A := crypto.Pem2Key(A_PrivKeyPem)
	B := crypto.Pem2Key(B_PrivKeyPem)
	B_Onion := crypto.GetOnionAddress(&B.PublicKey)

	// test successful challenge response
	challenge, r, err := GenerateChallenge(&A.PublicKey, &B.PublicKey)
	if err != nil {
		t.Fatalf("could not generate challenge: error: %s\n", err.Error())
	}

	response, err := GenerateResponse(challenge, A, B_Onion)
	if err != nil {
		t.Fatalf("could not generate response, error: %s\n", err.Error())
	}

	if response.R != r {
		t.Fatalf("response r (%x) and challenge r differ (%x)\n", response.R, r)
	}
}

func TestWrongResponse(t *testing.T) {
	A_PrivKeyPem := []byte(`-----BEGIN RSA PRIVATE KEY-----
MIICXAIBAAKBgQCxO9MKOEVPy/Y3+q6W33pQS4DwrO4k0zXaysnuliaREYASvoyF
P1l0ohffdyTuEUPz4MMLgoAtQeakwT2JIhPUVUae9BPzRFOl7ZSPVfNEWZxPnGMv
uePIklI0TACyPSByRZ+j7YvmKSOOXfyU0TiyB2PDxmQEPVxhxyy0w/GE0QIDAQAB
AoGAGAyzJXbfSOW3Yn88w7JNianFNGNy6UJT032jCyIK17KVO3Xp4YboH6CDNsqX
E0r6epRsQxqRRBLmNkMWk44xPGvgETyK3l23xNK6plKk6B6xONFo7b2D40C/Ids/
vw/IPtADgnJ+EXcN3pwbqA1UobKsh6K0QYmLhUzvqNH0xOUCQQDl2cHf/gkJiGfV
U2nP71QY1JMMNuMjYgQ2vVdXrg1vF2TVCQ+FgocnsC+Hn/9Qjd9AnlRmCte1k8k4
q7nQh+QTAkEAxWWiiZ5d0fLsf7qf+T3r8lOgC9G0AXXMZlYtWDlcMIjZJ90fY3HW
11N8BxcAgMqhvnIAmTSFhAolzL3mYSFoCwJAKR1ho7qiTTU8NZmdJNfPuD1WLGop
CSxASrZlyEZXGtcZb04Wm7A0kvHeHqmNFxC5rapkuHgaC93qsZtuOpIERwJBAL1Y
kGCRmE0bR9/9lBXwX7NCo/KyZIhCBp7javuFifjETAkBAmrRd9N0MTRbzA++Twfv
MsPCrY/KbBfI0IO7F6ECQFXdGVtPjYC+99JFbSoWlRj8nripWTmvlp3aYPD3ZAJU
RAczb2wbylOLxb3pidW9Tweo+kRXYYuXNFZ3vXnAjkE=
-----END RSA PRIVATE KEY-----`)

	B_PrivKeyPem := []byte(`-----BEGIN RSA PRIVATE KEY-----
MIICXwIBAAKBgQC6vjPYmMdzhnwLpmRvrUl6VhWDl+gAWOhmOFuVV9c8+EkQIqQH
g6E/0Y/+Tbj+aIiqUFxQ8vLdzXSrwR1XrkF8sUtF43QJaAGhMn5yPKawop4f/q+6
sBzomOHz+8t9yDJbC1NEJ6AcogGxPzga1CFCOptFjSzaAfay3I1G2nHMYwIDAQAB
AoGBAKraCiVI4nJXwHYILivepzM+P0C+YoyuyzF6ro/cZhhqMK6Kgvg8/fKdkNhh
07cvfJoWG+AT5w/3QZ9Cd84Yp+AUR4rJzSBhCHmmoTnZ1FGZWxNmfdoNn6pBvoz6
1wZtOd6pszLVce1YShv2AHLZr+rhaRe7BCcJDiEEGcX4sz4BAkEA62rjA9rPsElX
9fM5O9/aI2RdK5hqvgDFK9ntzb3BJ886l3Fi0/ifJjdiKI1nALRC54vYW4/PsECK
F+e2u6U9IQJBAMsR4eTtYHVik7NcUwRWNeensHEgQkS55+uCCQJIV+HLfcB8jcc3
2yoOml49On6JOZbPGxCi30te4a4GAs5SdQMCQQCyIkEBC7MX24eZbZ+jNLFlEm6F
rGEowIBxvAd7JNhhfScCrSNw7bHPQx0dPlHwcHYnquPd9KXc4hkcGZNlzZTBAkEA
jc4mBdwx4Kb+52BQZJXjPKqgDs9tF1sO9imvKtXj8LxOS01vIDAELuFVsPtmzpf7
DDIB/2MNNS/DvudZrERuiwJBAJ0PHZfkEYXaJb1bjYzHuc6o0gtggVcH2B6JrqRs
W90vDdM0H6z3hJdyrhPWcQxlkvrwVoXTsFvEQzZU2O7xSmo=
-----END RSA PRIVATE KEY-----`)

	A := crypto.Pem2Key(A_PrivKeyPem)
	B := crypto.Pem2Key(B_PrivKeyPem)
	B_Onion := crypto.GetOnionAddress(&B.PublicKey)

	// test response with wrong r
	challenge, _, err := GenerateChallenge(&A.PublicKey, &B.PublicKey)
	if err != nil {
		t.Fatalf("could not generate challenge: error: %s\n", err.Error())
	}

	wrongChallenge := Challenge{sha256.Sum256([]byte{1, 2, 3, 4}),
		challenge.PubKey, challenge.Enc}
	_, err = GenerateResponse(wrongChallenge, A, B_Onion)
	if err != nil {
		t.Logf("good, invalid r was detected: %s\n", err.Error())
	} else {
		t.Fatal("invalid r was not detected!")
	}
}
