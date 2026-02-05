func render(w http.ResponseWriter, r *http.Request, tmplName string, data interface{}, files ...string) {
	allFiles := append([]string{"ui/templates/layout.html"}, files...)
	tmpl, err := template.ParseFiles(allFiles...)
	if err != nil {
		http.Error(w, "Template Parse Error: "+err.Error(), http.StatusInternalServerError)
		return
	}

	token, _ := r.Context().Value(middleware.CSRFTokenKey).(string)

	wrapper := struct {
		Data      interface{}
		CSRFToken string
	}{
		Data:      data,
		CSRFToken: token,
	}

	if err := tmpl.ExecuteTemplate(w, "layout", wrapper); err != nil {
		http.Error(w, "Template Execute Error: "+err.Error(), http.StatusInternalServerError)
	}
}
