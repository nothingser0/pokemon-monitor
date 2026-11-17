package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"math/rand"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"
)

const maxEmbedsPerBatch = 10

const defaultPayload = `{"background_only":false,"blacklist_forms":[],"blacklist_pokemon_ids":[],"blacklist_species_forms":[],"boosted":null,"center_lat":null,"center_lng":null,"cp_max":null,"cp_min":null,"forms":[],"gender":null,"iv_mode":"percent","iv_percent_max":null,"iv_percent_min":100,"level_max":null,"level_min":null,"location":null,"min_ttl":null,"pokemon_ids":[585,828,660,33,505,480,481,482,201,849],"pvp_include_pre_evolutions":true,"pvp_league":"great","pvp_min_cp":null,"pvp_mode":false,"pvp_top_rank":null,"radius_km":null,"raw_iv_atk":null,"raw_iv_def":null,"raw_iv_sta":null,"shiny_account":null,"shiny_only":false,"sizes":[],"sort":"DSP","species_forms":[],"weather_id":null}`

type Config struct {
	APIURL      string          `json:"api_url"`
	IntervalMin int             `json:"interval_min"`
	IntervalMax int             `json:"interval_max"`
	Payload     json.RawMessage `json:"payload"`
}

type PokemonResult struct {
	EncounterID    json.RawMessage `json:"encounter_id"`
	PokemonID      json.RawMessage `json:"pokemon_id"`
	PokemonName    string          `json:"pokemon_name"`
	Gender         string          `json:"gender"`
	CP             float64         `json:"cp"`
	Level          float64         `json:"level"`
	IVPercent      float64         `json:"iv_percent"`
	RawIV          string          `json:"raw_iv"`
	Location       string          `json:"location"`
	RevealCode     string          `json:"reveal_code"`
	WeatherBoosted bool            `json:"weather_boosted"`
	FastMove       string          `json:"fast_move"`
	ChargeMove     string          `json:"charge_move"`
	DespawnAt      string          `json:"despawn_at"`
	Lat            float64         `json:"lat"`
	Lng            float64         `json:"lng"`
}

type APIResponse struct {
	Results      []PokemonResult `json:"results"`
	TotalMatches int             `json:"total_matches"`
	Tier         string          `json:"tier"`
}

type DiscordWebhookPayload struct {
	Content string         `json:"content,omitempty"`
	Embeds  []DiscordEmbed `json:"embeds,omitempty"`
}

type DiscordEmbed struct {
	Title     string        `json:"title"`
	Color     int           `json:"color"`
	Thumbnail *DiscordImage `json:"thumbnail,omitempty"`
	Fields    []EmbedField  `json:"fields"`
	Footer    *EmbedFooter  `json:"footer,omitempty"`
}

type DiscordImage struct {
	URL string `json:"url"`
}

type EmbedField struct {
	Name   string `json:"name"`
	Value  string `json:"value"`
	Inline bool   `json:"inline"`
}

type EmbedFooter struct {
	Text string `json:"text"`
}

var httpClient = &http.Client{
	Timeout: 30 * time.Second,
}

func loadEnvFile(path string) {
	data, err := os.ReadFile(path)
	if err != nil {
		return
	}
	lines := strings.Split(string(data), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		parts := strings.SplitN(line, "=", 2)
		if len(parts) == 2 {
			k := strings.TrimSpace(parts[0])
			v := strings.TrimSpace(parts[1])
			v = strings.Trim(v, `"'`)
			if _, exists := os.LookupEnv(k); !exists {
				_ = os.Setenv(k, v)
			}
		}
	}
}

func getEnv(key, fallback string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return fallback
}

func rawToString(raw json.RawMessage) string {
	s := strings.TrimSpace(string(raw))
	s = strings.Trim(s, `"`)
	return s
}

func buildConfig() Config {
	cfg := Config{
		APIURL:      getEnv("API_URL", "https://coordinates-api.pokemongopro.com/search"),
		IntervalMin: 30,
		IntervalMax: 120,
		Payload:     json.RawMessage(defaultPayload),
	}

	configFile := os.Getenv("CONFIG_FILE")
	if configFile == "" {
		configFile = "config.json"
	}
	if data, err := os.ReadFile(configFile); err == nil {
		var wrapper struct {
			APIURL      string          `json:"api_url"`
			IntervalMin int             `json:"interval_min"`
			IntervalMax int             `json:"interval_max"`
			Payload     json.RawMessage `json:"payload"`
		}
		if err := json.Unmarshal(data, &wrapper); err == nil && len(wrapper.Payload) > 0 {
			if wrapper.APIURL != "" {
				cfg.APIURL = wrapper.APIURL
			}
			if wrapper.IntervalMin > 0 {
				cfg.IntervalMin = wrapper.IntervalMin
			}
			if wrapper.IntervalMax > 0 {
				cfg.IntervalMax = wrapper.IntervalMax
			}
			cfg.Payload = wrapper.Payload
		} else {
			// If config.json directly contains the payload object
			var testMap map[string]any
			if err := json.Unmarshal(data, &testMap); err == nil && len(testMap) > 0 {
				cfg.Payload = json.RawMessage(data)
			}
		}
	}

	// Environment variable overrides
	if v := os.Getenv("PAYLOAD_JSON"); v != "" {
		cfg.Payload = json.RawMessage(v)
	}
	if v := os.Getenv("INTERVAL_MIN"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			cfg.IntervalMin = n
		}
	}
	if v := os.Getenv("INTERVAL_MAX"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			cfg.IntervalMax = n
		}
	}
	if v := os.Getenv("API_URL"); v != "" {
		cfg.APIURL = v
	}

	return cfg
}

func loadState(path string) map[string]bool {
	set := make(map[string]bool)
	data, err := os.ReadFile(path)
	if err != nil {
		return set
	}
	var ids []string
	if err := json.Unmarshal(data, &ids); err == nil {
		for _, id := range ids {
			set[id] = true
		}
	}
	return set
}

func saveState(path string, set map[string]bool) error {
	dir := filepath.Dir(path)
	if dir != "" && dir != "." {
		_ = os.MkdirAll(dir, 0755)
	}
	ids := make([]string, 0, len(set))
	for id := range set {
		ids = append(ids, id)
	}
	data, err := json.Marshal(ids)
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0644)
}

func fetchPokemon(apiURL, bearerToken string, payload json.RawMessage) (*APIResponse, error) {
	req, err := http.NewRequest(http.MethodPost, apiURL, bytes.NewReader(payload))
	if err != nil {
		return nil, fmt.Errorf("create request error: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+bearerToken)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/133.0.0.0 Safari/537.36")
	req.Header.Set("Accept", "application/json, text/plain, */*")
	req.Header.Set("Accept-Language", "en-US,en;q=0.9,id;q=0.8")
	req.Header.Set("Origin", "https://pogocoordinates.com")
	req.Header.Set("Referer", "https://pogocoordinates.com/")

	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request error: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		respBody, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API status %d: %s", resp.StatusCode, string(respBody))
	}

	var result APIResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decode error: %w", err)
	}

	return &result, nil
}

func parseDespawnTime(despawnStr string) *time.Time {
	if despawnStr == "" {
		return nil
	}

	formats := []string{
		time.RFC3339,
		time.RFC3339Nano,
		"2006-01-02T15:04:05Z07:00",
		"2006-01-02T15:04:05",
		"2006-01-02 15:04:05",
	}

	for _, f := range formats {
		if t, err := time.Parse(f, despawnStr); err == nil {
			return &t
		}
	}
	return nil
}

func buildCountdown(despawnStr string) string {
	t := parseDespawnTime(despawnStr)
	if t == nil {
		return "Unknown"
	}
	return fmt.Sprintf("<t:%d:R>", t.Unix())
}

func buildEmbed(p PokemonResult) DiscordEmbed {
	name := p.PokemonName
	if name == "" {
		name = "Unknown"
	}
	pokemonIDStr := rawToString(p.PokemonID)
	if pokemonIDStr == "" {
		pokemonIDStr = "?"
	}

	genderEmoji := ""
	if p.Gender == "Male" {
		genderEmoji = " ♂️"
	} else if p.Gender == "Female" {
		genderEmoji = " ♀️"
	}

	weatherEmoji := ""
	if p.WeatherBoosted {
		weatherEmoji = " ☀️"
	}

	color := 0x3498DB // Blue (normal)
	if p.IVPercent >= 90 {
		color = 0x00FF00 // Green (high IV)
	} else if p.IVPercent >= 80 {
		color = 0xFFAA00 // Orange (good IV)
	}

	revealURL := ""
	if p.RevealCode != "" {
		revealURL = "https://coordinates-api.pokemongopro.com/reveal/" + p.RevealCode
	} else {
		revealURL = "https://www.google.com/maps/search/?api=1&query=" + url.QueryEscape(p.Location)
	}

	imageURL := fmt.Sprintf("https://raw.githubusercontent.com/PokeAPI/sprites/master/sprites/pokemon/other/official-artwork/%s.png", pokemonIDStr)

	rawIV := p.RawIV
	if rawIV == "" {
		rawIV = "?/?/?"
	}

	fastMove := p.FastMove
	if fastMove == "" {
		fastMove = "Unknown"
	}
	chargeMove := p.ChargeMove
	if chargeMove == "" {
		chargeMove = "Unknown"
	}

	loc := p.Location
	if loc == "" {
		loc = "Unknown"
	}

	levelStr := fmt.Sprintf("%.1f", p.Level)
	if p.Level == math.Floor(p.Level) {
		levelStr = fmt.Sprintf("%.0f", p.Level)
	}
	cpStr := fmt.Sprintf("%.0f", p.CP)

	return DiscordEmbed{
		Title: fmt.Sprintf("%s%s%s", name, genderEmoji, weatherEmoji),
		Color: color,
		Thumbnail: &DiscordImage{
			URL: imageURL,
		},
		Fields: []EmbedField{
			{Name: "IV", Value: fmt.Sprintf("**%.1f%%** (%s)", p.IVPercent, rawIV), Inline: true},
			{Name: "Level", Value: fmt.Sprintf("**%s**", levelStr), Inline: true},
			{Name: "CP", Value: fmt.Sprintf("**%s**", cpStr), Inline: true},
			{Name: "Moves", Value: fmt.Sprintf("%s / %s", fastMove, chargeMove), Inline: false},
			{Name: "Despawns", Value: buildCountdown(p.DespawnAt), Inline: false},
			{Name: "Location", Value: fmt.Sprintf("[%s](%s)", loc, revealURL), Inline: false},
		},
		Footer: &EmbedFooter{
			Text: fmt.Sprintf("ID: %s", pokemonIDStr),
		},
	}
}

func sendDiscord(webhookURL string, pokemonList []PokemonResult) {
	if len(pokemonList) == 0 {
		return
	}

	limit := len(pokemonList)
	if limit > maxEmbedsPerBatch {
		limit = maxEmbedsPerBatch
	}
	batch := pokemonList[:limit]

	for _, p := range batch {
		embed := buildEmbed(p)
		payload := DiscordWebhookPayload{
			Embeds: []DiscordEmbed{embed},
		}

		body, err := json.Marshal(payload)
		if err != nil {
			continue
		}

		req, err := http.NewRequest(http.MethodPost, webhookURL, bytes.NewReader(body))
		if err != nil {
			continue
		}
		req.Header.Set("Content-Type", "application/json")

		resp, err := httpClient.Do(req)
		if err != nil {
			fmt.Fprintf(os.Stderr, "❌ Discord webhook error for %s: %v\n", p.PokemonName, err)
			continue
		}
		resp.Body.Close()

		name := p.PokemonName
		if name == "" {
			name = "Unknown"
		}
		fmt.Printf("✅ Notified: %s (#%s) at %s\n", name, rawToString(p.PokemonID), p.Location)
	}

	remaining := len(pokemonList) - limit
	if remaining > 0 {
		payload := DiscordWebhookPayload{
			Content: fmt.Sprintf("...%d lainnya", remaining),
		}
		if body, err := json.Marshal(payload); err == nil {
			if req, err := http.NewRequest(http.MethodPost, webhookURL, bytes.NewReader(body)); err == nil {
				req.Header.Set("Content-Type", "application/json")
				if resp, err := httpClient.Do(req); err == nil {
					resp.Body.Close()
				}
			}
		}
	}
}

func checkOnce(cfg Config, webhookURL, bearerToken, stateFile string) {
	fmt.Printf("🔍 Checking Pokemon API at %s\n", time.Now().Format("2006-01-02 15:04:05"))

	data, err := fetchPokemon(cfg.APIURL, bearerToken, cfg.Payload)
	if err != nil {
		fmt.Fprintf(os.Stderr, "❌ API error: %v\n", err)
		return
	}

	tier := data.Tier
	if tier == "" {
		tier = "unknown"
	}
	fmt.Printf("📊 Tier: %s | Matches: %d | Results: %d\n", tier, data.TotalMatches, len(data.Results))

	if len(data.Results) == 0 {
		fmt.Println("✅ No Pokemon found")
		return
	}

	notified := loadState(stateFile)
	var newPokemon []PokemonResult

	for _, p := range data.Results {
		uid := rawToString(p.EncounterID)
		if uid == "" {
			uid = fmt.Sprintf("%s_%.6f_%.6f", rawToString(p.PokemonID), p.Lat, p.Lng)
		}

		if !notified[uid] {
			newPokemon = append(newPokemon, p)
			notified[uid] = true
		}
	}

	if len(newPokemon) > 0 {
		sendDiscord(webhookURL, newPokemon)
		if err := saveState(stateFile, notified); err != nil {
			fmt.Fprintf(os.Stderr, "⚠️ Failed to save state: %v\n", err)
		}
		fmt.Printf("✅ %d new Pokemon detected and notified\n", len(newPokemon))
	} else {
		fmt.Println("ℹ️  All Pokemon already notified")
	}
}

func getRandomInterval(cfg Config) time.Duration {
	minMin := cfg.IntervalMin
	maxMin := cfg.IntervalMax
	if minMin <= 0 {
		minMin = 10
	}
	if maxMin < minMin {
		minMin, maxMin = maxMin, minMin
	}

	diff := maxMin - minMin
	var minutes int
	if diff == 0 {
		minutes = minMin
	} else {
		minutes = minMin + rand.Intn(diff+1)
	}
	return time.Duration(minutes) * time.Minute
}

func main() {
	envFile := getEnv("ENV_FILE", ".env")
	loadEnvFile(envFile)

	stateFile := getEnv("STATE_FILE", "data/.state.json")
	if _, err := os.Stat("data"); os.IsNotExist(err) && stateFile == "data/.state.json" {
		if _, err := os.Stat(".state.json"); err == nil {
			stateFile = ".state.json"
		}
	}

	if len(os.Args) > 1 && os.Args[1] == "--config" {
		fmt.Printf("📝 Env file: %s\n", envFile)
		fmt.Printf("📝 State file: %s\n", stateFile)
		return
	}

	runOnce := false
	for _, arg := range os.Args[1:] {
		if arg == "--once" {
			runOnce = true
			break
		}
	}

	webhookURL := os.Getenv("WEBHOOK_URL")
	bearerToken := os.Getenv("BEARER_TOKEN")

	if webhookURL == "" || bearerToken == "" {
		fmt.Fprintln(os.Stderr, "❌ Missing configuration. Please set WEBHOOK_URL and BEARER_TOKEN")
		fmt.Fprintf(os.Stderr, "   Edit %s (see .env.example for reference)\n", envFile)
		os.Exit(1)
	}

	cfg := buildConfig()

	if runOnce {
		checkOnce(cfg, webhookURL, bearerToken, stateFile)
		return
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	fmt.Println("🔄 Starting continuous monitoring (Ctrl+C to stop)")

	for {
		// Reload config dynamically so edits to config.json or environment take effect immediately
		currentCfg := buildConfig()
		checkOnce(currentCfg, webhookURL, bearerToken, stateFile)

		interval := getRandomInterval(currentCfg)
		fmt.Printf("--- Next check in %d minutes ---\n", int(interval.Minutes()))

		select {
		case <-ctx.Done():
			fmt.Println("\n👋 Stopped by user")
			return
		case <-time.After(interval):
		}
	}
}
