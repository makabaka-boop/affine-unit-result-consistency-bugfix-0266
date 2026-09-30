package main

import (
	"encoding/json"
	"errors"
	"flag"
	"log"
	"net/http"
	"os"

	"github.com/example/units/internal/unitcalc"
)

type evalRequest struct {
	Expression string `json:"expression"`
}

type errorResponse struct {
	Error    string `json:"error"`
	Kind     string `json:"kind"`
	SrcStart int    `json:"srcStart"`
	SrcEnd   int    `json:"srcEnd"`
	Locator  string `json:"locator"`
}

func writeEvalError(w http.ResponseWriter, src string, start, end int, kind, msg string) {
	if start < 0 {
		start = 0
	}
	if end > unitcalc.RuneLen(src) || end <= start {
		end = unitcalc.RuneLen(src)
	}
	resp := errorResponse{
		Error:    msg,
		Kind:     kind,
		SrcStart: start,
		SrcEnd:   end,
		Locator:  buildLocator(src, start, end),
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusBadRequest)
	_ = json.NewEncoder(w).Encode(resp)
}

// buildLocator renders the source with a caret under the offending span,
// e.g. "20 C * 2\n^^^^^^^". Offsets are rune-based, matching the service API.
func buildLocator(src string, start, end int) string {
	runes := []rune(src)
	if len(runes) == 0 {
		return ""
	}
	caret := make([]rune, len(runes))
	for i := range caret {
		caret[i] = ' '
	}
	for i := start; i < end && i < len(runes); i++ {
		caret[i] = '^'
	}
	return string(runes) + "\n" + string(caret)
}

func main() {
	addr := flag.String("addr", ":8080", "listen address")
	healthcheckURL := flag.String("healthcheck-url", "",
		"if set, GET this URL once and exit 0/1 instead of serving (container healthcheck)")
	flag.Parse()

	if *healthcheckURL != "" {
		resp, err := http.Get(*healthcheckURL)
		if err != nil {
			log.Printf("healthcheck failed: %v", err)
			os.Exit(1)
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			log.Printf("healthcheck: status %d", resp.StatusCode)
			os.Exit(1)
		}
		os.Exit(0)
	}

	mux := http.NewServeMux()

	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"status":"ok"}`))
	})

	mux.HandleFunc("GET /units", func(w http.ResponseWriter, r *http.Request) {
		type unitInfo struct {
			Symbol string `json:"symbol"`
			Kind   string `json:"kind"`
			Dim    string `json:"dim"`
		}
		out := []unitInfo{}
		for _, sym := range unitcalc.UnitSymbols() {
			u, _ := unitcalc.LookupUnit(sym)
			out = append(out, unitInfo{Symbol: sym, Kind: u.Kind.String(), Dim: u.Dim.String()})
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(out)
	})

	mux.HandleFunc("POST /api/eval", func(w http.ResponseWriter, r *http.Request) {
		var req evalRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeEvalError(w, "", 0, 0, "request", "request body must be JSON with an \"expression\" field")
			return
		}
		if req.Expression == "" {
			writeEvalError(w, req.Expression, 0, 0, "request", "expression is empty")
			return
		}
		res, err := unitcalc.Evaluate(req.Expression)
		if err != nil {
			var se *unitcalc.SyntaxError
			var te *unitcalc.TypedOpError
			switch {
			case errors.As(err, &se):
				writeEvalError(w, req.Expression, se.Start, se.End, "syntax", se.Msg)
			case errors.As(err, &te):
				writeEvalError(w, req.Expression, te.Start, te.End, "type", te.Msg)
			default:
				writeEvalError(w, req.Expression, 0, len(req.Expression), "eval", err.Error())
			}
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(res)
	})

	log.Printf("units service listening on %s", *addr)
	log.Fatal(http.ListenAndServe(*addr, mux))
}
