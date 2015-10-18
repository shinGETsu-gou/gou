/*
 * Copyright (c) 2015, Shinya Yagyu
 * All rights reserved.
 * Redistribution and use in source and binary forms, with or without
 * modification, are permitted provided that the following conditions are met:
 *
 * 1. Redistributions of source code must retain the above copyright notice,
 *    this list of conditions and the following disclaimer.
 * 2. Redistributions in binary form must reproduce the above copyright notice,
 *    this list of conditions and the following disclaimer in the documentation
 *    and/or other materials provided with the distribution.
 * 3. Neither the name of the copyright holder nor the names of its
 *    contributors may be used to endorse or promote products derived from this
 *    software without specific prior written permission.
 *
 * THIS SOFTWARE IS PROVIDED BY THE COPYRIGHT HOLDERS AND CONTRIBUTORS "AS IS"
 * AND ANY EXPRESS OR IMPLIED WARRANTIES, INCLUDING, BUT NOT LIMITED TO, THE
 * IMPLIED WARRANTIES OF MERCHANTABILITY AND FITNESS FOR A PARTICULAR PURPOSE
 * ARE DISCLAIMED. IN NO EVENT SHALL THE COPYRIGHT HOLDER OR CONTRIBUTORS BE
 * LIABLE FOR ANY DIRECT, INDIRECT, INCIDENTAL, SPECIAL, EXEMPLARY, OR
 * CONSEQUENTIAL DAMAGES (INCLUDING, BUT NOT LIMITED TO, PROCUREMENT OF
 * SUBSTITUTE GOODS OR SERVICES; LOSS OF USE, DATA, OR PROFITS; OR BUSINESS
 * INTERRUPTION) HOWEVER CAUSED AND ON ANY THEORY OF LIABILITY, WHETHER IN
 * CONTRACT, STRICT LIABILITY, OR TORT (INCLUDING NEGLIGENCE OR OTHERWISE)
 * ARISING IN ANY WAY OUT OF THE USE OF THIS SOFTWARE, EVEN IF ADVISED OF THE
 * POSSIBILITY OF SUCH DAMAGE.
 */

package gou

import (
	"log"
	"testing"
)

func TestApllo(t *testing.T) {
	pkey := makePrivateKey("test")
	log.Println(pkey.keyD, pkey.keyN)
	pub, pri := pkey.getKeys()
	log.Println(pkey.keyD, pkey.keyN)
	log.Println(pub, pri)
	if pub != "DpmzfQSOhbpxE7xuaiEao3ztv9NAJi/loTs2N43f5hC3XpT3z9VhApcrYy94XhMBKONo5H14c8STrriPJnCcVA" {
		t.Fatal("publickey key unmatch")
	}
	if pri != "BAcp0SUgUOSY+TrLhy/MEszzq0Obadi3EhXDEUUD9FmOkv7vhPiNrgg2HR8DmuFiPcXNHdqu44wyGRX5bmdcQA" {
		t.Fatal("privatekey unmatch")
	}
	log.Println(pkey.keyD, pkey.keyN)
	s := pkey.sign("test")
	log.Println(s)
	if s != "7peLqh1dbHjwmDpmREUytCu7k/2S3cS2eLYn+z42TaQkaHoyTRVUTKekbinRQQpkEGJah0hyDDIPc+AZHjecDA" {
		t.Fatalf("sign failed")
	}
	if v := verify("test", s, pub); !v {
		t.Fatalf("verify failed")
	}
}
