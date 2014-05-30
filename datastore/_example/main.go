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

package main

import (
	"net/http"
	"time"

	"appengine"

	"upper.io/db"
	_ "upper.io/db/datastore"
)

var settings = db.Settings{}

type Birthday struct {
	// Maps the "Name" property to the "name" column of the "birthdays" table.
	Name string `bson:"name"`
	// Maps the "Born" property to the "born" column of the "birthdays" table.
	Born time.Time `bson:"born"`
}

func Example(rw http.ResponseWriter, req *http.Request) {
	c := appengine.NewContext(req)
	sess, _ := db.Open("datastore", db.Settings{Context: c})

	birthdayCollection, err := sess.Collection("birthdays")

	if err != nil {
		if err != db.ErrCollectionDoesNotExists {
			c.Errorf("Could not use collection: %q\n", err)
		}
	} else {
		err = birthdayCollection.Truncate()

		if err != nil {
			c.Errorf("Truncate(): %q\n", err)
		}
	}

	// Inserting some rows into the "birthdays" table.

	birthdayCollection.Append(Birthday{
		Name: "Hayao Miyazaki",
		Born: time.Date(1941, time.January, 5, 0, 0, 0, 0, time.UTC),
	})

	birthdayCollection.Append(Birthday{
		Name: "Nobuo Uematsu",
		Born: time.Date(1959, time.March, 21, 0, 0, 0, 0, time.UTC),
	})

	birthdayCollection.Append(Birthday{
		Name: "Hironobu Sakaguchi",
		Born: time.Date(1962, time.November, 25, 0, 0, 0, 0, time.UTC),
	})

	// Let's query for the results we've just inserted.
	res := birthdayCollection.Find()

	var birthdays []Birthday

	// Query all results and fill the birthdays variable with them.
	err = res.All(&birthdays)

	if err != nil {
		c.Errorf("res.All(): %q\n", err)
	}

	// Printing to stdout.
	for _, birthday := range birthdays {
		c.Infof("%s was born in %s.\n", birthday.Name, birthday.Born.Format("January 2, 2006"))
	}
}

func init() {

	http.HandleFunc("/", Example)

}
