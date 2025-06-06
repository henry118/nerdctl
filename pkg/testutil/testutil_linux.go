/*
   Copyright The containerd Authors.

   Licensed under the Apache License, Version 2.0 (the "License");
   you may not use this file except in compliance with the License.
   You may obtain a copy of the License at

       http://www.apache.org/licenses/LICENSE-2.0

   Unless required by applicable law or agreed to in writing, software
   distributed under the License is distributed on an "AS IS" BASIS,
   WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
   See the License for the specific language governing permissions and
   limitations under the License.
*/

package testutil

var (
	BusyboxImage                = "ghcr.io/containerd/busybox:1.36"
	AlpineImage                 = mirrorOf("alpine:3.13")
	NginxAlpineImage            = mirrorOf("nginx:1.19-alpine")
	NginxAlpineIndexHTMLSnippet = "<title>Welcome to nginx!</title>"
	RegistryImageStable         = mirrorOf("registry:2")
	RegistryImageNext           = "ghcr.io/distribution/distribution:"
	WordpressImage              = mirrorOf("wordpress:5.7")
	WordpressIndexHTMLSnippet   = "<title>WordPress &rsaquo; Installation</title>"
	MariaDBImage                = mirrorOf("mariadb:10.5")
	DockerAuthImage             = mirrorOf("cesanta/docker_auth:1.7")
	FluentdImage                = "fluent/fluentd:v1.17.0-debian-1.0"
	KuboImage                   = mirrorOf("ipfs/kubo:v0.16.0")
	SystemdImage                = "ghcr.io/containerd/stargz-snapshotter:0.15.1-kind"
	GolangImage                 = mirrorOf("golang:1.18")

	// Source: https://gist.github.com/cpuguy83/fcf3041e5d8fb1bb5c340915aabeebe0
	NonDistBlobImage = "ghcr.io/cpuguy83/non-dist-blob:latest"
	// Foreign layer digest
	NonDistBlobDigest = "sha256:be691b1535726014cdf3b715ff39361b19e121ca34498a9ceea61ad776b9c215"

	CommonImage = AlpineImage

	// This error string is expected when attempting to connect to a TCP socket
	// for a service which actively refuses the connection.
	// (e.g. attempting to connect using http to an https endpoint).
	// It should be "connection refused" as per the TCP RFC.
	// https://www.rfc-editor.org/rfc/rfc793
	ExpectedConnectionRefusedError = "connection refused"
)

const (
	FedoraESGZImage = "ghcr.io/stargz-containers/fedora:30-esgz"            // eStargz
	FfmpegSociImage = "public.ecr.aws/soci-workshop-examples/ffmpeg:latest" // SOCI
	UbuntuImage     = "public.ecr.aws/docker/library/ubuntu:23.10"          // Large enough for testing soci index creation
)
