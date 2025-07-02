package web

import (
	"github.com/hazcod/shade/pkg/storage"
	"github.com/sirupsen/logrus"
	"html/template"
	"net/http"
)

var dashboardTmpl = template.Must(template.New("dashboard").Parse(`
<html>
<head><title>Dashboard</title></head>
<body>
	<h1>SaaS in use:</h1>
	<ul>
		{{range .Domains}}
			<li>{{.}}</li>
		{{else}}
			<li>No domains found.</li>
		{{end}}
	</ul>

	<h1>Duplicate Passwords:</h1>
	<ul>
		{{range $user, $hashes := .DuplicatePasswords}}
			<li>
				<strong>{{$user}}</strong>
				<ul>
					{{range $hash, $domains := $hashes}}
						<li>Domains: {{$domains}}</li>
					{{end}}
				</ul>
			</li>
		{{else}}
			<li>No duplicate passwords found.</li>
		{{end}}
	</ul>
</body>
</html>
`))

type dashboardData struct {
	Domains            []string
	DuplicatePasswords map[string]map[string]string
}

func GetDashboard(logger *logrus.Logger, store storage.Driver) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
			return
		}

		domains, err := store.GetAllDomains()
		if err != nil {
			logger.WithError(err).Error("error getting all domains")
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}

		dupePasswords, err := store.GetDuplicatePasswords()
		if err != nil {
			logger.WithError(err).Error("error getting duplicate passwords")
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}

		data := dashboardData{
			Domains:            domains,
			DuplicatePasswords: dupePasswords,
		}

		w.Header().Set("Content-Type", "text/html")
		if err := dashboardTmpl.Execute(w, data); err != nil {
			logger.WithError(err).Error("error rendering template")
			http.Error(w, "Template Error", http.StatusInternalServerError)
		}
	}
}
