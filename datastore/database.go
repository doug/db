// +build appengine

/*
  Copyright (c) 2014-2016 Doug Fritz, https://dougfritz.com

  Permission is hereby granted, free of charge, to any person obtaining
  a copy of this software and associated documentation files (the
  "Software"), to deal in the Software without restriction, including
  without limitation the rights to use, copy, modify, merge, publish,
  distribute, sublicense, and/or sell copies of the Software, and to
  permit persons to whom the Software is furnished to do so, subject to
  the following conditions:

  The above copyright notice and this permission notice shall be
  included in all copies or substantial portions of the Software.

  THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND,
  EXPRESS OR IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF
  MERCHANTABILITY, FITNESS FOR A PARTICULAR PURPOSE AND
  NONINFRINGEMENT. IN NO EVENT SHALL THE AUTHORS OR COPYRIGHT HOLDERS BE
  LIABLE FOR ANY CLAIM, DAMAGES OR OTHER LIABILITY, WHETHER IN AN ACTION
  OF CONTRACT, TORT OR OTHERWISE, ARISING FROM, OUT OF OR IN CONNECTION
  WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE SOFTWARE.
*/

package datastore

import (
	"log"
	"os"
	"strings"

	"appengine"

	"upper.io/db"
)

const (
	driverName = `datastore`
)

type Source struct {
	config db.Settings
}

func debugEnabled() bool {
	if os.Getenv(db.EnvEnableDebug) != "" {
		return true
	}
	return false
}

func init() {
	db.Register(driverName, &Source{})
}

func debugLogQuery(c *chunks) {
	log.Printf("Fields: %v\nLimit: %v\nOffset: %v\nSort: %v\nConditions: %v\n", c.Fields, c.Limit, c.Offset, c.Sort, c.Conditions)
}

// Returns the string name of the database.
func (self *Source) Name() string {
	return self.config.Name
}

// Get the current Context
func (self *Source) Context() appengine.Context {
	if c, ok := self.config.Context.(appengine.Context); ok {
		return c
	}
	return nil
}

// Stores database settings.
func (self *Source) Setup(config db.Settings) error {
	self.config = config
	return self.Open()
}

// Returns the underlying datastore instance.
func (self *Source) Driver() interface{} {
	return ds
}

// Open a connection. Not neccessary.
func (self *Source) Open() error {
	return nil
}

// Closes the current database session. Resets to default namespace.
func (self *Source) Close() error {
	if self.session != nil {
		self.session.Close()
	}
	return nil
}

// Changes the active database. Default "". Using Namespaces https://developers.google.com/appengine/docs/go/reference#Namespace
func (self *Source) Use(database string) (err error) {
	self.config.Database = database
	self.config.Context, err = appengine.Namespace(self.Context(), database)
	return
}

// Starts a transaction block.
func (self *Source) Begin() error {
	return nil
}

// Ends a transaction block.
func (self *Source) End() error {
	return nil
}

// Drops the currently active database. (NOT SUPPORTED)
func (self *Source) Drop() error {
	return db.ErrFeatureNotSupported
}

// Returns a slice of non-system collection names within the active
// database.
func (self *Source) Collections() (cols []string, err error) {
	var rawcols []string
	var col string

	if rawcols, err = self.database.CollectionNames(); err != nil {
		return nil, err
	}

	cols = make([]string, 0, len(rawcols))

	for _, col = range rawcols {
		if strings.HasPrefix(col, "system.") == false {
			cols = append(cols, col)
		}
	}

	return cols, nil
}

// Returns a collection instance by name.
func (self *Source) Collection(name string) (db.Collection, error) {
	var err error

	col := &Collection{}
	col.parent = self
	col.collection = self.database.C(name)

	col.DB = self
	col.SetName = name

	if col.Exists() == false {
		err = db.ErrCollectionDoesNotExists
	}

	return col, err
}
