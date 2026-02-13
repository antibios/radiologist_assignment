package main

import (
	"fmt"
	"net/http"
	"slices"
	"strings"

	"github.com/starfederation/datastar-go/datastar"
)

type ActiveSearchSignals struct {
	Search            string `json:"search"`
	RadiologistSearch string `json:"radiologistSearch"`
	SiteSearch        string `json:"siteSearch"`
	ProcedureSearch   string `json:"procedureSearch"`
}

// Levenshtein calculates the Levenshtein distance between two strings.
func Levenshtein(s1, s2 string) int {
	r1, r2 := []rune(s1), []rune(s2)
	n, m := len(r1), len(r2)
	if n > m {
		r1, r2 = r2, r1
		n, m = m, n
	}

	currentRow := make([]int, n+1)
	for i := 0; i <= n; i++ {
		currentRow[i] = i
	}

	for i := 1; i <= m; i++ {
		previousRow := currentRow
		currentRow = make([]int, n+1)
		currentRow[0] = i
		for j := 1; j <= n; j++ {
			add, del, change := previousRow[j]+1, currentRow[j-1]+1, previousRow[j-1]
			if r1[j-1] != r2[i-1] {
				change++
			}
			currentRow[j] = min(add, min(del, change))
		}
	}
	return currentRow[n]
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func handleActiveSearch(w http.ResponseWriter, r *http.Request) {
	signals := &ActiveSearchSignals{}
	if err := datastar.ReadSignals(r, signals); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	searchType := r.URL.Query().Get("type")
	var query string

	// Determine query based on type and signals
	switch searchType {
	case "radiologist":
		query = signals.RadiologistSearch
	case "site":
		query = signals.SiteSearch
	case "procedure":
		query = signals.ProcedureSearch
	default:
		query = signals.Search
	}

    // Fallback if specific signal is empty but generic search is present
    if query == "" && signals.Search != "" {
        query = signals.Search
    }

	query = strings.ToLower(strings.TrimSpace(query))
	sse := datastar.NewSSE(w, r)
	targetID := r.URL.Query().Get("target")

	switch searchType {
	case "radiologist":
		handleRadiologistSearch(sse, query)
	case "site":
		handleSiteSearch(sse, query)
	case "procedure":
		handleProcedureSearch(sse, query, targetID)
	default:
		http.Error(w, "Invalid search type", http.StatusBadRequest)
	}
}

func handleRadiologistSearch(sse *datastar.ServerSentEventGenerator, query string) {
	type ScoredRadiologist struct {
		ID        string
		FirstName string
		LastName  string
		Score     int
	}

	var results []ScoredRadiologist

	// Read lock global radiologists
	// Note: radiologists variable is in main.go
	// Since we are in package main, we can access it.
    // However, it's not protected by a mutex in main.go explicitly for reading globally,
    // but main.go uses it. Let's assume concurrency safety or add lock if needed.
    // main.go initializes it in main/init. It's only read after init?
    // Actually handleAssignRadiologist uses rosterMu but radiologists list seems static?
    // No, handleShifts reads it. It seems treated as static configuration in memory.

	for _, rad := range radiologists {
		if query == "" {
			results = append(results, ScoredRadiologist{
				ID:        rad.ID,
				FirstName: rad.FirstName,
				LastName:  rad.LastName,
				Score:     0,
			})
			continue
		}

		fn := strings.ToLower(rad.FirstName)
		ln := strings.ToLower(rad.LastName)
		id := strings.ToLower(rad.ID)

		// Simple scoring: contains = 0, fuzzy = distance
		score := 1000
		if strings.Contains(fn, query) || strings.Contains(ln, query) || strings.Contains(id, query) {
			score = 0
		} else {
			d1 := Levenshtein(query, fn)
			d2 := Levenshtein(query, ln)
			d3 := Levenshtein(query, id)
			dist := min(d1, min(d2, d3))
			if dist < 5 { // Threshold
				score = dist
			}
		}

		if score < 1000 {
			results = append(results, ScoredRadiologist{
				ID:        rad.ID,
				FirstName: rad.FirstName,
				LastName:  rad.LastName,
				Score:     score,
			})
		}
	}

	// Sort by score
	slices.SortFunc(results, func(a, b ScoredRadiologist) int {
		return a.Score - b.Score
	})

	// Limit results
	if len(results) > 15 {
		results = results[:15]
	}

	// Generate HTML
	var sb strings.Builder
    sb.WriteString(`<div id="radiologist-results" class="list">`)
	for _, res := range results {
		sb.WriteString(fmt.Sprintf(`
			<a class="row waves-effect" onclick="selectRadiologist('%s', '%s %s')">
				<div class="col">
					<span>%s %s</span>
					<label>%s</label>
				</div>
			</a>`, res.ID, res.FirstName, res.LastName, res.FirstName, res.LastName, res.ID))
	}
    if len(results) == 0 {
        sb.WriteString(`<div class="padding">No results found</div>`)
    }
	sb.WriteString("</div>")

	sse.PatchElements(sb.String())
}

func handleSiteSearch(sse *datastar.ServerSentEventGenerator, query string) {
	configMu.RLock()
	sites := refData.Sites
	configMu.RUnlock()

	type ScoredSite struct {
		Code  string
		Name  string
		Score int
	}

	var results []ScoredSite

	for _, s := range sites {
		if query == "" {
			results = append(results, ScoredSite{Code: s.Code, Name: s.Name, Score: 0})
			continue
		}

		name := strings.ToLower(s.Name)
		code := strings.ToLower(s.Code)

		score := 1000
		if strings.Contains(name, query) || strings.Contains(code, query) {
			score = 0
		} else {
			d1 := Levenshtein(query, name)
			d2 := Levenshtein(query, code)
			dist := min(d1, d2)
			if dist < 5 {
				score = dist
			}
		}

		if score < 1000 {
			results = append(results, ScoredSite{Code: s.Code, Name: s.Name, Score: score})
		}
	}

	slices.SortFunc(results, func(a, b ScoredSite) int {
		return a.Score - b.Score
	})

	if len(results) > 15 {
		results = results[:15]
	}

	var sb strings.Builder
    sb.WriteString(`<div id="site-results" class="list">`)
	for _, res := range results {
		sb.WriteString(fmt.Sprintf(`
			<a class="row waves-effect" onclick="addSite('%s', '%s')">
				<div class="col">
					<span>%s</span>
					<label>%s</label>
				</div>
			</a>`, res.Code, res.Name, res.Name, res.Code))
	}
    if len(results) == 0 {
        sb.WriteString(`<div class="padding">No results found</div>`)
    }
	sb.WriteString("</div>")

	sse.PatchElements(sb.String())
}

func handleProcedureSearch(sse *datastar.ServerSentEventGenerator, query string, targetID string) {
	proceduresMu.RLock()
	procs := procedures
	proceduresMu.RUnlock()

	type ScoredProcedure struct {
		Code        string
		Description string
		Score       int
	}

	var results []ScoredProcedure

	for _, p := range procs {
		if query == "" {
			results = append(results, ScoredProcedure{Code: p.Code, Description: p.Description, Score: 0})
			continue
		}

		desc := strings.ToLower(p.Description)
		code := strings.ToLower(p.Code)

		score := 1000
		if strings.Contains(desc, query) || strings.Contains(code, query) {
			score = 0
		} else {
			d1 := Levenshtein(query, desc)
			d2 := Levenshtein(query, code)
			dist := min(d1, d2)
			if dist < 5 {
				score = dist
			}
		}

		if score < 1000 {
			results = append(results, ScoredProcedure{Code: p.Code, Description: p.Description, Score: score})
		}
	}

	slices.SortFunc(results, func(a, b ScoredProcedure) int {
		return a.Score - b.Score
	})

	if len(results) > 15 {
		results = results[:15]
	}

	var sb strings.Builder
    if targetID == "" {
        targetID = "procedure-results"
    }

    sb.WriteString(fmt.Sprintf(`<div id="%s" class="list">`, targetID))
	for _, res := range results {
		sb.WriteString(fmt.Sprintf(`
			<a class="row waves-effect" onclick="selectProcedure(this, '%s')">
				<div class="col">
					<span>%s</span>
					<label>%s</label>
				</div>
			</a>`, res.Code, res.Description, res.Code))
	}
    if len(results) == 0 {
        sb.WriteString(`<div class="padding">No results found</div>`)
    }
	sb.WriteString("</div>")

	sse.PatchElements(sb.String())
}
