package swaggerui

import (
	"embed"
	"errors"
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
	// Получаем доступ к встроенным статическим файлам
	staticFS, _ := fs.Sub(swagfs, "embed")

	// Создаем новый мультиплексор (мини-роутер) для нашего хендлера.
	// Это позволяет легко разделить обработку /swagger_spec и статики.
	mux := http.NewServeMux()

	// 1. Регистрируем обработчик для файла спецификации OpenAPI
	specPath := "/" + "spec"
	mux.HandleFunc(specPath, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write(spec)
	})

	// 2. Регистрируем обработчик для всех остальных путей (статика)
	staticHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// http.StripPrefix уже удалил базовый путь,
		// поэтому r.URL.Path содержит путь к файлу относительно корня Swagger UI.
		filePath := path.Clean(strings.TrimPrefix(r.URL.Path, "/"))

		// Если путь пустой (запрос к корню, например /docs/),
		// то отдаем index.html
		if filePath == "" || filePath == "." {
			filePath = "index.html"
		}

		file, err := staticFS.Open(filePath)
		if err != nil {
			// Если файл не найден (например, при обновлении страницы в SPA),
			// снова пытаемся отдать index.html. Это стандартная практика для SPA.
			if errors.Is(err, fs.ErrNotExist) {
				file, err = staticFS.Open("index.html")
				if err != nil {
					http.NotFound(w, r) // Если даже index.html нет, то 404
					return
				}
			} else {
				http.Error(w, "Internal Server Error", http.StatusInternalServerError)
				return
			}
		}
		defer file.Close()

		info, err := file.Stat()
		if err != nil {
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}

		// Используем http.ServeContent для корректной обработки заголовков
		// (Content-Type, ETag, Last-Modified и т.д.)
		http.ServeContent(w, r, info.Name(), info.ModTime(), file.(io.ReadSeeker))
	})

	mux.Handle("/", staticHandler)

	return mux
}
