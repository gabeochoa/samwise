package main

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/jinzhu/gorm"
	_ "github.com/jinzhu/gorm/dialects/sqlite"
	_ "github.com/mattn/go-sqlite3"
)

// Database Models
// They come with ID, CreatedAt, UpdatedAt and DeletedAt for free

// Folder model for ORM
type Folder struct {
	gorm.Model
	Name string
}

// Record model for ORM
type Record struct {
	gorm.Model
	FolderID int
	Folder   Folder
	Key      string
	Data     []byte
}

func initDB() {
	db, err := gorm.Open("sqlite3", "test.db")
	if err != nil {
		panic("failed to connect database")
	}
	defer db.Close()

	// Migrate the schema
	db.AutoMigrate(&Record{})
	db.AutoMigrate(&Folder{})

	// Apply extra rules

	// TODO: if we want cascase on delete folder
	//db.Model(&Profile{}).AddForeignKey("record_refer", "users(refer)", "CASCADE", "CASCADE")
}

// TODO: this could be a one liner...
func fileExists(path string) bool {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return false
	}
	return true
}

func getDBHandler() *gorm.DB {
	// TODO: always migrate until we have data we want to save
	os.Remove("test.db")

	// Check to see if we have a db already
	// TODO: right now we have to delete to migrate (i think)
	if !fileExists("test.db") {
		fmt.Println("TestDB doesnt exist, creating and migrating")
		initDB()
	}
	db, err := gorm.Open("sqlite3", "test.db")
	if err != nil {
		panic("failed to connect database")
	}
	return db
}

func insertFolder(dbHandle *gorm.DB, folder *Folder) {
	// TODO: check if already exists
	dbHandle.Create(folder)
	// TODO Check if it created successfully
}

func insertRecord(dbHandle *gorm.DB, record *Record) {
	dbHandle.Create(record)
}

func main() {

	db := getDBHandler()
	defer db.Close()
	// Create
	var exampleRecord = &Record{
		Folder: Folder{
			Name: "test",
		},
		Key:  "example",
		Data: []byte(`{"Name":"Bob","Food":"Pickle"}`),
	}
	insertRecord(db, exampleRecord)

	// TODO check searching
	// db.First(&product, "code = ?", "L1212") // find product with code l1212

	// Find the folder metadata associated with this
	var assocfolder Folder
	db.Model(exampleRecord).Related(&assocfolder)
	folderJ, _ := json.Marshal(assocfolder)
	fmt.Println("///Associated Folder ///")
	fmt.Println(string(folderJ))
	fmt.Println("///End Folder ///")
	// Update - update product's price to 2000
	// db.Model(&product).Update("Price", 2000)

	// Delete - delete product
	// db.Delete(ptrrecord)
	// db.Delete(ptrfolder)
}
