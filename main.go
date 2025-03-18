package main

import (
	"database/sql"
	"encoding/json"
	"html/template"
	"log"
	"net/http"
	"sort"
	"strconv"
	"strings"

	_ "modernc.org/sqlite"
)

type Collection struct {
	Id    int
	Name  string
	Items []Item
}

type Item struct {
	Id    int
	Image string
	Title string
	Wins  int
}

var tpl *template.Template
var db *sql.DB

// var validPath = regexp.MustCompile(`^/(edit|save|view)/([a-zA-Z0-9]+)(\?.*)?$`)

func initDB() {
	const setupQuery = `
	CREATE TABLE IF NOT EXISTS collections (
		id INTEGER NOT NULL PRIMARY KEY,
		name TEXT NOT NULL
	);
	CREATE TABLE IF NOT EXISTS items (
		id INTEGER NOT NULL PRIMARY KEY,
		image TEXT,
		title TEXT NOT NULL,
		wins INTEGER DEFAULT 0,
		collection_id INTEGER,
		FOREIGN KEY (collection_id) REFERENCES collections(id) ON DELETE CASCADE
	);`

	var err error
	db, err = sql.Open("sqlite", "test.db")
	if err != nil {
		log.Fatalf("initDB: %v", err)
	}

	_, err = db.Exec(setupQuery)
	if err != nil {
		log.Fatalf("initDB: %v", err)
	}
}

func parseTemplates() {
	tmp := template.New("")

	funcs := template.FuncMap{
		"embed": func(name string, data any) template.HTML {
			var out strings.Builder
			if err := tmp.ExecuteTemplate(&out, name, data); err != nil {
				log.Fatalf("parseTemplates: %v", err)
			}
			return template.HTML(out.String())
		}}

	tpl = template.Must(tmp.Funcs(funcs).ParseGlob("tmpl/*.html"))
}

// match the regexp

func main() {
	initDB()
	defer db.Close()

	err := db.Ping()
	if err != nil {
		log.Fatalf("Cannot connect to database: %v", err)
	}

	parseTemplates()

	http.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("static/"))))

	http.HandleFunc("/", homeHandler)
	http.HandleFunc("/calluponthecreator", creationHandler)
	http.HandleFunc("/calluponthecreator/create-item", createItemHandler)
	http.HandleFunc("/collections/", collectionsHandler)
	http.HandleFunc("/banthisguy", deleteItemHandler)
	http.HandleFunc("/reset/", resetHandler)
	http.HandleFunc("/delether", demColHandler)

	log.Fatal(http.ListenAndServe(":8080", nil))

}

func demColHandler(w http.ResponseWriter, r *http.Request) {
	collectionId, _ := strconv.Atoi(r.FormValue("collection_id"))
	db.Exec("DELETE FROM collections WHERE id = ?", collectionId)

	http.Redirect(w, r, "/", http.StatusSeeOther)
}

func resetHandler(w http.ResponseWriter, r *http.Request) {
	collectionIdStr := strings.Split(r.URL.Path, "/")[2]
	collectionId, _ := strconv.Atoi(collectionIdStr)

	db.Exec("UPDATE items SET wins = 0 WHERE collection_id = ?", collectionId)
	http.Redirect(w, r, "/"+collectionIdStr, http.StatusSeeOther)
}

func homeHandler(w http.ResponseWriter, r *http.Request) {
	collectionsRows, err := db.Query("SELECT id, name FROM collections")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer collectionsRows.Close()

	var collections []Collection

	for collectionsRows.Next() {
		var col Collection
		if err := collectionsRows.Scan(&col.Id, &col.Name); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		itemsRows, err := db.Query("SELECT id, image, title FROM items WHERE collection_id = ?", col.Id)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		defer itemsRows.Close()

		for itemsRows.Next() {
			var item Item
			if err := itemsRows.Scan(&item.Id, &item.Image, &item.Title); err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			col.Items = append(col.Items, item)
		}

		collections = append(collections, col)
	}

	// remove line
	parseTemplates()

	err = tpl.ExecuteTemplate(w, "base.html", map[string]any{
		"Page":   "home.html",
		"Script": "/static/js/home.js",
		"Data":   collections,
	})
	if err != nil {
		log.Println("Template error:", err)
		return
	}
}

func creationHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Har har~ We differ in methodology but could we still be friends?", http.StatusMethodNotAllowed)
	}
	name := r.FormValue("name") // skipped validation
	var id int
	err := db.QueryRow("INSERT INTO collections(name) VALUES(?) RETURNING id", name).Scan(&id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	http.Redirect(w, r, "/"+strconv.Itoa(id), http.StatusSeeOther)
}

func createItemHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Error due to provision of fictional methodology", http.StatusMethodNotAllowed)
	}
	collectionIdString := r.FormValue("collection_id")
	collectionId, err := strconv.Atoi(collectionIdString)
	if err != nil {
		log.Print(collectionId)
		http.Error(w, "Invalid collection ID", http.StatusBadRequest)
		return
	} // missing db call

	image := r.FormValue("image")
	title := r.FormValue("title")

	_, err = db.Exec("INSERT INTO items (image, title, collection_id) VALUES (?, ?, ?)", image, title, collectionId)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	http.Redirect(w, r, "/"+collectionIdString, http.StatusSeeOther)
}

func deleteItemHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Not very Methodic", http.StatusMethodNotAllowed)
	}

	collectionId := r.FormValue("collection_id") // should get from url
	itemId, _ := strconv.Atoi(r.FormValue("item_id"))

	db.Exec("DELETE FROM items WHERE id = ?", itemId) // validation

	http.Redirect(w, r, "/"+collectionId, http.StatusSeeOther)
}

func collectionsHandler(w http.ResponseWriter, r *http.Request) {
	if strings.Split(r.URL.Path, "/")[3] == "result" {
		collectionResultHandler(w, r)
		return
	}

	collectionId, err := strconv.Atoi(strings.Split(r.URL.Path, "/")[2])
	if err != nil {
		http.Error(w, "Invalid collection ID", http.StatusBadRequest)
		return
	} // missing db call

	var col Collection
	_ = db.QueryRow("SELECT id, name FROM collections WHERE id = ?", collectionId).Scan(&col.Id, &col.Name) // skipped validation

	itemRows, _ := db.Query("SELECT id, image, title FROM items WHERE collection_id = ?", collectionId) // skipped validation

	for itemRows.Next() {
		var item Item
		if err := itemRows.Scan(&item.Id, &item.Image, &item.Title); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
		col.Items = append(col.Items, item)
	}

	// remove line
	parseTemplates()

	err = tpl.ExecuteTemplate(w, "base.html", map[string]any{
		"Page":   "rank.html",
		"Script": "/static/js/rank.js",
		"Data":   col,
	})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

func collectionResultHandler(w http.ResponseWriter, r *http.Request) {
	collectionId := strings.Split(r.URL.Path, "/")[2]

	var scores map[string]int

	_ = json.Unmarshal([]byte(r.FormValue("scores")), &scores)

	tx, err := db.Begin()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
	defer tx.Rollback()

	for itemIdStr, scoreWins := range scores {
		itemId, _ := strconv.Atoi(itemIdStr)

		tx.Exec(
			"UPDATE items SET wins = wins + ?  WHERE id = ? AND collection_id = ?",
			scoreWins,
			itemId,
			collectionId,
		)

	}
	tx.Commit()

	var Res struct {
		Id     int
		Name   string
		User   []Item
		Public []Item
	}
	_ = db.QueryRow("SELECT id, name FROM collections WHERE id = ?", collectionId).Scan(&Res.Id, &Res.Name)

	itemRows, err := db.Query("SELECT id, image, title, wins FROM items WHERE collection_id = ?", collectionId)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer itemRows.Close()

	var items []Item
	var userItems []Item

	for itemRows.Next() {
		var item Item
		if err := itemRows.Scan(&item.Id, &item.Image, &item.Title, &item.Wins); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}

		items = append(items, item)

		item.Wins = scores[strconv.Itoa(item.Id)]

		userItems = append(userItems, item)
	}

	sort.Slice(items, func(i, j int) bool {
		return items[i].Wins > items[j].Wins
	})

	sort.Slice(userItems, func(i, j int) bool {
		return userItems[i].Wins > userItems[j].Wins
	})

	Res.Public, Res.User = items, userItems

	// remove line
	parseTemplates()

	err = tpl.ExecuteTemplate(w, "base.html", map[string]any{
		"Page": "result.html",
		"Data": Res,
	})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}
