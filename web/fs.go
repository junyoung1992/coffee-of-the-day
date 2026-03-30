package web

import (
	"embed"
	"io/fs"
	"net/http"
)

// static/ 디렉토리에 React 빌드 결과물을 embed한다.
// 빌드 전에는 .gitkeep만 포함되어 있어 컴파일 오류 없이 서버가 기동된다.
//
//go:embed all:static
var staticFS embed.FS

// Handler는 React SPA를 서빙하는 HTTP 핸들러를 반환한다.
// 정적 파일이 존재하면 그대로 서빙하고, 없으면 index.html로 fallback해
// React Router가 클라이언트 사이드 라우팅을 처리하도록 한다.
func Handler() http.Handler {
	sub, err := fs.Sub(staticFS, "static")
	if err != nil {
		panic(err)
	}

	fileServer := http.FileServer(http.FS(sub))

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path
		if path == "/" || path == "" {
			fileServer.ServeHTTP(w, r)
			return
		}

		// 파일이 실제로 존재하는지 확인한다.
		// 존재하지 않으면 React Router 경로이므로 index.html로 fallback한다.
		f, err := sub.Open(path[1:]) // 앞의 '/' 제거
		if err == nil {
			f.Close()
			fileServer.ServeHTTP(w, r)
			return
		}

		r2 := *r
		r2.URL.Path = "/"
		fileServer.ServeHTTP(w, &r2)
	})
}
