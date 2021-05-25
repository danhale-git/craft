module github.com/danhale-git/craft

go 1.16

require (
	github.com/Microsoft/go-winio v0.4.16 // indirect
	github.com/containerd/containerd v1.4.3 // indirect
	github.com/docker/distribution v2.7.1+incompatible // indirect
	github.com/docker/docker v20.10.0+incompatible
	github.com/docker/go-connections v0.4.0
	github.com/docker/go-units v0.4.0 // indirect
	github.com/fastly/go-utils v0.0.0-20180712184237-d95a45783239 // indirect
	github.com/gorilla/mux v1.8.0 // indirect
	github.com/jehiah/go-strftime v0.0.0-20171201141054-1d33003b3869 // indirect
	github.com/lestrrat/go-envload v0.0.0-20180220120943-6ed08b54a570 // indirect
	github.com/lestrrat/go-file-rotatelogs v0.0.0-20180223000712-d3151e2a480f
	github.com/lestrrat/go-strftime v0.0.0-20180220042222-ba3bf9c1d042 // indirect
	github.com/mitchellh/go-homedir v1.1.0
	github.com/moby/term v0.0.0-20201110203204-bea5bbe245bf
	github.com/morikuni/aec v1.0.0 // indirect
	github.com/opencontainers/go-digest v1.0.0 // indirect
	github.com/opencontainers/image-spec v1.0.1
	github.com/spf13/cobra v1.1.1
	github.com/tebeka/strftime v0.1.5 // indirect
	gotest.tools/v3 v3.0.3 // indirect
)

// https://github.com/containerd/containerd/issues/4703
//replace github.com/pkg/errors v0.8.1 => github.com/pkg/errors v0.9.1
