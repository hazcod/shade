package web

import (
	"github.com/hazcod/shade/pkg/storage"
	"github.com/sirupsen/logrus"
	"html/template"
	"net/http"
)

var dashboardTmpl = template.Must(template.New("dashboard").Parse(`
<!DOCTYPE html>
<html lang="en" data-bs-theme="auto">
<head>
    <meta charset="utf-8">
	<title>Dashboard</title>
	<link href="https://cdn.jsdelivr.net/npm/bootstrap@5.3.7/dist/css/bootstrap.min.css" rel="stylesheet" integrity="sha384-LN+7fdVzj6u52u30Kp6M/trliBMCMKTyK833zpbD+pXdCLuTusPj697FH4R/5mcr" crossorigin="anonymous">
	<script src="https://cdn.jsdelivr.net/npm/bootstrap@5.3.7/dist/js/bootstrap.bundle.min.js" integrity="sha384-ndDqU0Gzau9qJ1lfW4pNLlhNTkCfHzAVBReH9diLvGRem5+R9g2FzA8ZGN954O5Q" crossorigin="anonymous"></script>
</head>
<body>
	<div class="container">
		<article class="my-3">
			<div class="bd-heading sticky-xl-top align-self-start mt-5 mb-3 mt-xl-0 mb-xl-2">
				<h3>SaaS in use:</h3>
			</div>
			<div>
				<ul>
					{{range .Domains}}
						<li>{{.}}</li>
					{{else}}
						<li>No domains found.</li>
					{{end}}
				</ul>
			</div>

			<div class="bd-heading sticky-xl-top align-self-start mt-5 mb-3 mt-xl-0 mb-xl-2">
				<h3>Duplicate Passwords:</h3>
			</div>
			<div>
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
			</div>

		</article>
	</div>
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
