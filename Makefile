dev: 
	templ generate --watch --proxy="http://localhost:3000" --cmd="go run ./cmd/web" -open-browser=false

png-to-ico:
	magick -gravity center ./assets/lambda.png -flatten -colors 256 -background transparent ./assets/lambda.ico
