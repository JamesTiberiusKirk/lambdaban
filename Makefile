dev: 
	templ generate --include-version=false --watch --proxy="http://localhost:3030" --cmd="go run ./cmd/web" -open-browser=false

png-to-ico:
	magick -gravity center ./assets/lambda.png -flatten -colors 256 -background transparent ./assets/lambda.ico

docker-build:
	docker buildx build -t lambdaban -f ./cmd/web/Dockerfile .
