RELEASE_VERSION=0.1
build:
	docker build -t blanche -f Dockerfile.dev .

run:
	@touch .env
	docker run --rm -it -p 3000:3000 \
		-v `pwd`:/go/src/github.com/RentTheRunway/blanche \
		--env-file=.env \
		blanche bin/blanche "./*.go ./**/**/*.go"

.PHONY: build run
