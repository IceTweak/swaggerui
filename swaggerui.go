package swaggerui

import (
	"embed"
	"io"
	"io/fs"
	"net/http"
	"path"
	"strings"
)

//go:generate go run generate.go
//go:embed embed
var swagfs embed.FS

// Handler возвращает http.Handler, который обслуживает Swagger UI.
// Этот обработчик не зависит от пути и предназначен для использования
// с http.StripPrefix, чтобы его можно было смонтировать на любой URL.
func Handler(spec []byte) http.Handler {
	staticFS, _ := fs.Sub(swagfs, "embed")

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Очищаем путь
		urlPath := path.Clean(r.URL.Path)

		// Убираем ведущий слеш для работы с embedded FS
		filePath := strings.TrimPrefix(urlPath, "/")

		// Обработка специального пути для спецификации
		if filePath == "spec" {
			w.Header().Set("Content-Type", "application/json")
			w.Write(spec)
			return
		}

		// Если путь пустой, отдаем index.html
		if filePath == "" {
			filePath = "index.html"
		}

		// Пытаемся открыть запрашиваемый файл
		file, err := staticFS.Open(filePath)
		if err != nil {
			// Если файл не найден и это не JS/CSS, пытаемся отдать index.html (для SPA)
			if !strings.HasSuffix(filePath, ".js") &&
				!strings.HasSuffix(filePath, ".css") &&
				!strings.HasSuffix(filePath, ".png") {
				file, err = staticFS.Open("index.html")
				if err != nil {
					http.NotFound(w, r)
					return
				}
			} else {
				http.NotFound(w, r)
				return
			}
		}
		defer file.Close()

		// Получаем информацию о файле
		stat, err := file.Stat()
		if err != nil {
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}

		// Устанавливаем правильные заголовки Content-Type
		switch {
		case strings.HasSuffix(filePath, ".js"):
			w.Header().Set("Content-Type", "application/javascript")
		case strings.HasSuffix(filePath, ".css"):
			w.Header().Set("Content-Type", "text/css")
		case strings.HasSuffix(filePath, ".html"):
			w.Header().Set("Content-Type", "text/html")
		case strings.HasSuffix(filePath, ".png"):
			w.Header().Set("Content-Type", "image/png")
		}

		// Отдаем файл
		http.ServeContent(w, r, stat.Name(), stat.ModTime(), file.(io.ReadSeeker))
	})
}
