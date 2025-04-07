package middleware

// type templHandler struct {
// }
//
// type customTemplWriter struct {
// 	w          http.ResponseWriter
// 	statusCode int
// }
//
// func (c *customTemplWriter) Header() http.Header {
// 	return c.w.Header()
// }
//
// func (c *customTemplWriter) Write(b []byte) (int, error) {
// 	return c.w.Write(b)
// }
//
// func (c *customTemplWriter) WriteHeader(statusCode int) {
// 	c.statusCode = statusCode
// 	c.w.WriteHeader(statusCode)
// }
//
// func Templ(log *slog.Logger, next http.Handler) http.Handler {
// 	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
// 		start := time.Now()
//
// 		cw := &customTemplWriter{w: w}
//
// 		next.ServeHTTP(cw, r)
//
// 		log.Info("Request",
// 			"status", cw.statusCode,
// 			"method", r.Method,
// 			"uri", r.RequestURI,
// 			"remoteAddr", r.RemoteAddr,
// 			"userAgent", r.UserAgent(),
// 			"duration", time.Since(start),
// 		)
// 	})
// }
