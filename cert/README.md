# package cert

This package contains functions for creating a local development root
certificate that can then be used with net/http tlsConfig servers to dynamically
generate SSL certificates in development.

The generated certificates are kept in-memory in a cache and are re-generated
on next execution.

On macOS the certificate is installed into the system keychain (a password
prompt is shown) so all generated certificates are trusted automatically.

## Usage example

You need to run the root certificate creation step once:

```
  err := cert.CreateCert()
```

Now you can create an HTTPS server that uses the root certificate to generate
valid certificates dynamically:

```
  cache, err := cert.NewCache()

  tlsConfig := &tls.Config{
    GetCertificate: cache.GetCertificate,
  }
```

See https://github.com/moomerman/zap/tree/master/cert/example/main.go for
a full example.

## Credits

The majority of the code for this package was extracted from
https://github.com/puma/puma-dev.

```
Copyright (c) 2016, Evan Phoenix
All rights reserved.

Redistribution and use in source and binary forms, with or without
modification, are permitted provided that the following conditions are met:

* Redistributions of source code must retain the above copyright notice, this
  list of conditions and the following disclaimer.
* Redistributions in binary form must reproduce the above copyright notice
  this list of conditions and the following disclaimer in the documentation
  and/or other materials provided with the distribution.
* Neither the name of the Evan Phoenix nor the names of its contributors
  may be used to endorse or promote products derived from this software
  without specific prior written permission.

THIS SOFTWARE IS PROVIDED BY THE COPYRIGHT HOLDERS AND CONTRIBUTORS "AS IS"
AND ANY EXPRESS OR IMPLIED WARRANTIES, INCLUDING, BUT NOT LIMITED TO, THE
IMPLIED WARRANTIES OF MERCHANTABILITY AND FITNESS FOR A PARTICULAR PURPOSE ARE
DISCLAIMED. IN NO EVENT SHALL THE COPYRIGHT OWNER OR CONTRIBUTORS BE LIABLE
FOR ANY DIRECT, INDIRECT, INCIDENTAL, SPECIAL, EXEMPLARY, OR CONSEQUENTIAL
DAMAGES (INCLUDING, BUT NOT LIMITED TO, PROCUREMENT OF SUBSTITUTE GOODS OR
SERVICES; LOSS OF USE, DATA, OR PROFITS; OR BUSINESS INTERRUPTION) HOWEVER
CAUSED AND ON ANY THEORY OF LIABILITY, WHETHER IN CONTRACT, STRICT LIABILITY,
OR TORT (INCLUDING NEGLIGENCE OR OTHERWISE) ARISING IN ANY WAY OUT OF THE USE
OF THIS SOFTWARE, EVEN IF ADVISED OF THE POSSIBILITY OF SUCH DAMAGE.
```
