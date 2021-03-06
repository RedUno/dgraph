/*
 * Copyright 2019 Dgraph Labs, Inc. and Contributors
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package schema

import (
	"encoding/json"
	"io/ioutil"
	"strings"
	"testing"

	"github.com/dgraph-io/dgraph/graphql/authorization"
	"github.com/google/go-cmp/cmp"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/require"
	"github.com/vektah/gqlparser/v2/ast"
	"gopkg.in/yaml.v2"
)

func TestDgraphMapping_WithoutDirectives(t *testing.T) {
	schemaStr := `
type Author {
        id: ID!

        name: String! @search(by: [hash, trigram])
        dob: DateTime @search
        reputation: Float @search
        posts: [Post!] @hasInverse(field: author)
}

type Post {
        postID: ID!
        postType: PostType @search
        author: Author! @hasInverse(field: posts)
}

enum PostType {
        Fact
        Question
        Opinion
}

interface Employee {
        ename: String!
}

interface Character {
        id: ID!
        name: String! @search(by: [exact])
        appearsIn: [Episode!] @search
}

type Human implements Character & Employee {
        starships: [Starship]
        totalCredits: Float
}

type Droid implements Character {
        primaryFunction: String
}

enum Episode {
        NEWHOPE
        EMPIRE
        JEDI
}

type Starship {
        id: ID!
        name: String! @search(by: [term])
        length: Float
}`

	schHandler, errs := NewHandler(schemaStr)
	require.NoError(t, errs)
	sch, err := FromString(schHandler.GQLSchema())
	require.NoError(t, err)

	s, ok := sch.(*schema)
	require.True(t, ok, "expected to be able to convert sch to internal schema type")

	author := map[string]string{
		"name":       "Author.name",
		"dob":        "Author.dob",
		"reputation": "Author.reputation",
		"posts":      "Author.posts",
	}
	post := map[string]string{
		"postType": "Post.postType",
		"author":   "Post.author",
	}
	character := map[string]string{
		"name":      "Character.name",
		"appearsIn": "Character.appearsIn",
	}
	human := map[string]string{
		"ename":        "Employee.ename",
		"name":         "Character.name",
		"appearsIn":    "Character.appearsIn",
		"starships":    "Human.starships",
		"totalCredits": "Human.totalCredits",
	}
	droid := map[string]string{
		"name":            "Character.name",
		"appearsIn":       "Character.appearsIn",
		"primaryFunction": "Droid.primaryFunction",
	}
	starship := map[string]string{
		"name":   "Starship.name",
		"length": "Starship.length",
	}

	expected := map[string]map[string]string{
		"Author":              author,
		"UpdateAuthorPayload": author,
		"DeleteAuthorPayload": author,
		"Post":                post,
		"UpdatePostPayload":   post,
		"DeletePostPayload":   post,
		"Employee": map[string]string{
			"ename": "Employee.ename",
		},
		"Character":              character,
		"UpdateCharacterPayload": character,
		"DeleteCharacterPayload": character,
		"Human":                  human,
		"UpdateHumanPayload":     human,
		"DeleteHumanPayload":     human,
		"Droid":                  droid,
		"UpdateDroidPayload":     droid,
		"DeleteDroidPayload":     droid,
		"Starship":               starship,
		"UpdateStarshipPayload":  starship,
		"DeleteStarshipPayload":  starship,
	}

	if diff := cmp.Diff(expected, s.dgraphPredicate); diff != "" {
		t.Errorf("dgraph predicate map mismatch (-want +got):\n%s", diff)
	}
}

func TestDgraphMapping_WithDirectives(t *testing.T) {
	schemaStr := `
	type Author @dgraph(type: "dgraph.author") {
			id: ID!

			name: String! @search(by: [hash, trigram])
			dob: DateTime @search
			reputation: Float @search
			posts: [Post!] @hasInverse(field: author)
	}

	type Post @dgraph(type: "dgraph.Post") {
			postID: ID!
			postType: PostType @search @dgraph(pred: "dgraph.post_type")
			author: Author! @hasInverse(field: posts) @dgraph(pred: "dgraph.post_author")
	}

	enum PostType {
			Fact
			Question
			Opinion
	}

	interface Employee @dgraph(type: "dgraph.employee.en") {
			ename: String!
	}

	interface Character @dgraph(type: "performance.character") {
			id: ID!
			name: String! @search(by: [exact])
			appearsIn: [Episode!] @search @dgraph(pred: "appears_in")
	}

	type Human implements Character & Employee {
			starships: [Starship]
			totalCredits: Float @dgraph(pred: "credits")
	}

	type Droid implements Character @dgraph(type: "roboDroid") {
			primaryFunction: String
	}

	enum Episode {
			NEWHOPE
			EMPIRE
			JEDI
	}

	type Starship @dgraph(type: "star.ship") {
			id: ID!
			name: String! @search(by: [term]) @dgraph(pred: "star.ship.name")
			length: Float
	}`

	schHandler, errs := NewHandler(schemaStr)
	require.NoError(t, errs)
	sch, err := FromString(schHandler.GQLSchema())
	require.NoError(t, err)

	s, ok := sch.(*schema)
	require.True(t, ok, "expected to be able to convert sch to internal schema type")

	author := map[string]string{
		"name":       "dgraph.author.name",
		"dob":        "dgraph.author.dob",
		"reputation": "dgraph.author.reputation",
		"posts":      "dgraph.author.posts",
	}
	post := map[string]string{
		"postType": "dgraph.post_type",
		"author":   "dgraph.post_author",
	}
	character := map[string]string{
		"name":      "performance.character.name",
		"appearsIn": "appears_in",
	}
	human := map[string]string{
		"ename":        "dgraph.employee.en.ename",
		"name":         "performance.character.name",
		"appearsIn":    "appears_in",
		"starships":    "Human.starships",
		"totalCredits": "credits",
	}
	droid := map[string]string{
		"name":            "performance.character.name",
		"appearsIn":       "appears_in",
		"primaryFunction": "roboDroid.primaryFunction",
	}
	starship := map[string]string{
		"name":   "star.ship.name",
		"length": "star.ship.length",
	}

	expected := map[string]map[string]string{
		"Author":              author,
		"UpdateAuthorPayload": author,
		"DeleteAuthorPayload": author,
		"Post":                post,
		"UpdatePostPayload":   post,
		"DeletePostPayload":   post,
		"Employee": map[string]string{
			"ename": "dgraph.employee.en.ename",
		},
		"Character":              character,
		"UpdateCharacterPayload": character,
		"DeleteCharacterPayload": character,
		"Human":                  human,
		"UpdateHumanPayload":     human,
		"DeleteHumanPayload":     human,
		"Droid":                  droid,
		"UpdateDroidPayload":     droid,
		"DeleteDroidPayload":     droid,
		"Starship":               starship,
		"UpdateStarshipPayload":  starship,
		"DeleteStarshipPayload":  starship,
	}

	if diff := cmp.Diff(expected, s.dgraphPredicate); diff != "" {
		t.Errorf("dgraph predicate map mismatch (-want +got):\n%s", diff)
	}
}

func TestCheckNonNulls(t *testing.T) {

	gqlSchema, err := FromString(`
	type T {
		req: String!
		notReq: String
		alsoReq: String!
	}`)
	require.NoError(t, err)

	tcases := map[string]struct {
		obj map[string]interface{}
		exc string
		err error
	}{
		"all present": {
			obj: map[string]interface{}{"req": "here", "notReq": "here", "alsoReq": "here"},
			err: nil,
		},
		"only non-null": {
			obj: map[string]interface{}{"req": "here", "alsoReq": "here"},
			err: nil,
		},
		"missing non-null": {
			obj: map[string]interface{}{"req": "here", "notReq": "here"},
			err: errors.Errorf("type T requires a value for field alsoReq, but no value present"),
		},
		"missing all non-null": {
			obj: map[string]interface{}{"notReq": "here"},
			err: errors.Errorf("type T requires a value for field req, but no value present"),
		},
		"with exclusion": {
			obj: map[string]interface{}{"req": "here", "notReq": "here"},
			exc: "alsoReq",
			err: nil,
		},
	}

	typ := &astType{
		typ:      &ast.Type{NamedType: "T"},
		inSchema: (gqlSchema.(*schema)),
	}

	for name, test := range tcases {
		t.Run(name, func(t *testing.T) {
			err := typ.EnsureNonNulls(test.obj, test.exc)
			if test.err == nil {
				require.NoError(t, err)
			} else {
				require.EqualError(t, err, test.err.Error())
			}
		})
	}
}

func TestSubstituteVarsInBody(t *testing.T) {
	tcases := []struct {
		name        string
		variables   map[string]interface{}
		template    interface{}
		expected    interface{}
		expectedErr error
	}{
		{
			"handle nil template correctly",
			map[string]interface{}{"id": "0x3", "postID": "0x9"},
			nil,
			nil,
			nil,
		},
		{
			"handle empty object template correctly",
			map[string]interface{}{"id": "0x3", "postID": "0x9"},
			map[string]interface{}{},
			map[string]interface{}{},
			nil,
		},
		{
			"substitutes variables correctly",
			map[string]interface{}{"id": "0x3", "postID": "0x9"},
			map[string]interface{}{"author": "$id", "post": map[string]interface{}{"id": "$postID"}},
			map[string]interface{}{"author": "0x3", "post": map[string]interface{}{"id": "0x9"}},
			nil,
		},
		{
			"substitutes nil variables correctly",
			map[string]interface{}{"id": nil},
			map[string]interface{}{"author": "$id"},
			map[string]interface{}{"author": nil},
			nil,
		},
		{
			"substitutes variables with an array in template correctly",
			map[string]interface{}{"id": "0x3", "admin": false, "postID": "0x9",
				"text": "Random comment", "age": 28},
			map[string]interface{}{"author": "$id", "admin": "$admin",
				"post": map[string]interface{}{"id": "$postID",
					"comments": []interface{}{"$text", "$age"}}, "age": "$age"},
			map[string]interface{}{"author": "0x3", "admin": false,
				"post": map[string]interface{}{"id": "0x9",
					"comments": []interface{}{"Random comment", 28}}, "age": 28},
			nil,
		},
		{
			"substitutes variables with an array of object in template correctly",
			map[string]interface{}{"id": "0x3", "admin": false, "postID": "0x9",
				"text": "Random comment", "age": 28},
			map[string]interface{}{"author": "$id", "admin": "$admin",
				"post": map[string]interface{}{"id": "$postID",
					"comments": []interface{}{map[string]interface{}{"text": "$text"}}}, "age": "$age"},
			map[string]interface{}{"author": "0x3", "admin": false,
				"post": map[string]interface{}{"id": "0x9",
					"comments": []interface{}{map[string]interface{}{"text": "Random comment"}}}, "age": 28},
			nil,
		},
		{
			"substitutes array variables correctly",
			map[string]interface{}{"ids": []int{1, 2, 3}, "names": []string{"M1", "M2"},
				"check": []interface{}{1, 3.14, "test"}},
			map[string]interface{}{"ids": "$ids", "names": "$names", "check": "$check"},
			map[string]interface{}{"ids": []int{1, 2, 3}, "names": []string{"M1", "M2"},
				"check": []interface{}{1, 3.14, "test"}},
			nil,
		},
		{
			"substitutes object variables correctly",
			map[string]interface{}{"author": map[string]interface{}{"id": 1, "name": "George"}},
			map[string]interface{}{"author": "$author"},
			map[string]interface{}{"author": map[string]interface{}{"id": 1, "name": "George"}},
			nil,
		},
		{
			"substitutes array of object variables correctly",
			map[string]interface{}{"authors": []interface{}{map[string]interface{}{"id": 1,
				"name": "George"}, map[string]interface{}{"id": 2, "name": "Jerry"}}},
			map[string]interface{}{"authors": "$authors"},
			map[string]interface{}{"authors": []interface{}{map[string]interface{}{"id": 1,
				"name": "George"}, map[string]interface{}{"id": 2, "name": "Jerry"}}},
			nil,
		},
		{
			"substitutes direct body variable correctly",
			map[string]interface{}{"authors": []interface{}{map[string]interface{}{"id": 1,
				"name": "George"}, map[string]interface{}{"id": 2, "name": "Jerry"}}},
			"$authors",
			[]interface{}{map[string]interface{}{"id": 1, "name": "George"},
				map[string]interface{}{"id": 2, "name": "Jerry"}},
			nil,
		},
		{
			"variable not found error",
			map[string]interface{}{"postID": "0x9"},
			map[string]interface{}{"author": "$id", "post": map[string]interface{}{"id": "$postID"}},
			nil,
			errors.New("couldn't find variable: $id in variables map"),
		},
	}

	for _, test := range tcases {
		t.Run(test.name, func(t *testing.T) {
			var templatePtr *interface{}
			if test.template == nil {
				templatePtr = nil
			} else {
				templatePtr = &test.template
			}
			err := SubstituteVarsInBody(templatePtr, test.variables)
			if test.expectedErr == nil {
				require.NoError(t, err)
				require.Equal(t, test.expected, test.template)
			} else {
				require.EqualError(t, err, test.expectedErr.Error())
			}
		})
	}
}

func TestParseBodyTemplate(t *testing.T) {
	tcases := []struct {
		name           string
		template       string
		expected       interface{}
		requiredFields map[string]bool
		expectedErr    error
	}{
		{
			"parses empty body template correctly",
			``,
			nil,
			nil,
			nil,
		},
		{
			"parses whitespace body template correctly",
			` `,
			nil,
			nil,
			nil,
		},
		{
			"parses empty object body template correctly",
			`{}`,
			map[string]interface{}{},
			map[string]bool{},
			nil,
		},
		{
			"parses body template correctly",
			`{ author: $id, post: { id: $postID }}`,
			map[string]interface{}{"author": "$id", "post": map[string]interface{}{"id": "$postID"}},
			map[string]bool{"id": true, "postID": true},
			nil,
		},
		{
			"parses body template with an array correctly",
			`{ author: $id, admin: $admin, post: { id: $postID, comments: [$text] },
			   age: $age}`,
			map[string]interface{}{"author": "$id", "admin": "$admin",
				"post": map[string]interface{}{"id": "$postID",
					"comments": []interface{}{"$text"}}, "age": "$age"},
			map[string]bool{"id": true, "admin": true, "postID": true, "text": true, "age": true},
			nil,
		},
		{
			"parses body template with an array of object correctly",
			`{ author: $id, admin: $admin, post: { id: $postID, comments: [{ text: $text }] },
			   age: $age}`,
			map[string]interface{}{"author": "$id", "admin": "$admin",
				"post": map[string]interface{}{"id": "$postID",
					"comments": []interface{}{map[string]interface{}{"text": "$text"}}}, "age": "$age"},
			map[string]bool{"id": true, "admin": true, "postID": true, "text": true, "age": true},
			nil,
		},
		{
			"parses body template with direct variable correctly",
			`$authors`,
			"$authors",
			map[string]bool{"authors": true},
			nil,
		},
		{
			"json unmarshal error",
			`{ author: $id, post: { id $postID }}`,
			nil,
			nil,
			errors.New("couldn't unmarshal HTTP body: {\"author\":\"$id\",\"post\":{\"id\"\"$postID\"}}" +
				" as JSON"),
		},
		{
			"unmatched brackets error",
			`{{ author: $id, post: { id: $postID }}`,
			nil,
			nil,
			errors.New("found unmatched curly braces while parsing body template"),
		},
		{
			"invalid character error",
			`(author: $id, post: { id: $postID }}`,
			nil,
			nil,
			errors.New("invalid character: ( while parsing body template"),
		},
	}

	for _, test := range tcases {
		t.Run(test.name, func(t *testing.T) {
			b, requiredFields, err := parseBodyTemplate(test.template)
			if test.expectedErr == nil {
				require.NoError(t, err)
				require.Equal(t, test.requiredFields, requiredFields)
				if b == nil {
					require.Nil(t, test.expected)
				} else {
					require.Equal(t, test.expected, *b)
				}
			} else {
				require.EqualError(t, err, test.expectedErr.Error())
			}
		})
	}
}

func TestSubstituteVarsInURL(t *testing.T) {
	tcases := []struct {
		name        string
		variables   map[string]interface{}
		url         string
		expected    string
		expectedErr error
	}{
		{
			"Return url as is when no params",
			nil,
			"http://myapi.com/favMovies/1?num=10",
			"http://myapi.com/favMovies/1?num=10",
			nil,
		},
		{
			"Substitute query params with space properly",
			map[string]interface{}{"id": "0x9", "name": "Michael Compton",
				"num": 10},
			"http://myapi.com/favMovies/$id?name=$name&num=$num",
			"http://myapi.com/favMovies/0x9?name=Michael+Compton&num=10",
			nil,
		},
		{
			"Substitute query params for variables with array value",
			map[string]interface{}{"ids": []int{1, 2}, "names": []string{"M1", "M2"},
				"check": []interface{}{1, 3.14, "test"}},
			"http://myapi.com/favMovies?id=$ids&name=$names&check=$check",
			"http://myapi.com/favMovies?check=1&check=3.14&check=test&id=1&id=2&name=M1&name=M2",
			nil,
		},
		{
			"Substitute query params for variables with object value",
			map[string]interface{}{"data": map[string]interface{}{"id": 1, "name": "George"}},
			"http://myapi.com/favMovies?author=$data",
			"http://myapi.com/favMovies?author%5Bid%5D=1&author%5Bname%5D=George",
			nil,
		},
		{
			"Substitute query params for variables with array of object value",
			map[string]interface{}{"data": []interface{}{map[string]interface{}{"id": 1,
				"name": "George"}, map[string]interface{}{"id": 2, "name": "Jerry"}}},
			"http://myapi.com/favMovies?author=$data",
			"http://myapi.com/favMovies?author%5Bid%5D=1&author%5Bid%5D=2&author%5Bname%5D=George" +
				"&author%5Bname%5D=Jerry",
			nil,
		},
		{
			"Substitute query params for a variable value that is null as empty",
			map[string]interface{}{"id": "0x9", "name": nil, "num": 10},
			"http://myapi.com/favMovies/$id?name=$name&num=$num",
			"http://myapi.com/favMovies/0x9?name=&num=10",
			nil,
		},
		{
			"Remove query params corresponding to variables that are empty.",
			map[string]interface{}{"id": "0x9", "num": 10},
			"http://myapi.com/favMovies/$id?name=$name&num=$num",
			"http://myapi.com/favMovies/0x9?num=10",
			nil,
		},
		{
			"Substitute multiple path params properly",
			map[string]interface{}{"id": "0x9", "num": 10},
			"http://myapi.com/favMovies/$id/$num",
			"http://myapi.com/favMovies/0x9/10",
			nil,
		},
		{
			"Substitute path params for variables with array value",
			map[string]interface{}{"ids": []int{1, 2}, "names": []string{"M1", "M2"},
				"check": []interface{}{1, 3.14, "test"}},
			"http://myapi.com/favMovies/$ids/$names/$check",
			"http://myapi.com/favMovies/1%2C2/M1%2CM2/1%2C3.14%2Ctest",
			nil,
		},
		{
			"Substitute path params for variables with object value",
			map[string]interface{}{"author": map[string]interface{}{"id": 1, "name": "George"}},
			"http://myapi.com/favMovies/$author",
			"http://myapi.com/favMovies/id%2C1%2Cname%2CGeorge",
			nil,
		},
		{
			"Substitute path params for variables with array of object value",
			map[string]interface{}{"authors": []interface{}{map[string]interface{}{"id": 1,
				"name": "George/"}, map[string]interface{}{"id": 2, "name": "Jerry"}}},
			"http://myapi.com/favMovies/$authors",
			"http://myapi.com/favMovies/id%2C1%2Cname%2CGeorge%2F%2Cid%2C2%2Cname%2CJerry",
			nil,
		},
	}

	for _, test := range tcases {
		t.Run(test.name, func(t *testing.T) {
			b, err := SubstituteVarsInURL(test.url, test.variables)
			if test.expectedErr == nil {
				require.NoError(t, err)
				require.Equal(t, test.expected, string(b))
			} else {
				require.EqualError(t, err, test.expectedErr.Error())
			}
		})
	}
}

func TestParseRequiredArgsFromGQLRequest(t *testing.T) {
	tcases := []struct {
		name         string
		req          string
		body         string
		requiredArgs map[string]bool
	}{
		{
			"parse required args for single request",
			"query($id: ID!, $age: String!) { userNames(id: $id, age: $age) }",
			"",
			map[string]bool{"id": true, "age": true},
		},
		{
			"parse required nested args for single request",
			"query($id: ID!, $age: String!) { userNames(id: $id, car: {age: $age}) }",
			"",
			map[string]bool{"id": true, "age": true},
		},
	}

	for _, test := range tcases {
		t.Run(test.name, func(t *testing.T) {
			args, err := parseRequiredArgsFromGQLRequest(test.req)
			require.NoError(t, err)
			require.Equal(t, test.requiredArgs, args)
		})
	}
}

// Tests showing that the correct query and variables are sent to the remote server.
type CustomHTTPConfigCase struct {
	Name string
	Type string

	// the query and variables given as input by the user.
	GQLQuery     string
	GQLVariables string
	// our schema against which the above query and variables are resolved.
	GQLSchema string

	// for resolving fields variables are populated from the result of resolving a Dgraph query
	// so RemoteVariables won't have anything.
	InputVariables string
	// remote query and variables which are built as part of the HTTP config and checked.
	RemoteQuery     string
	RemoteVariables string
	// remote schema against which the RemoteQuery and RemoteVariables are validated.
	RemoteSchema string
}

func TestGraphQLQueryInCustomHTTPConfig(t *testing.T) {
	b, err := ioutil.ReadFile("custom_http_config_test.yaml")
	require.NoError(t, err, "Unable to read test file")

	var tests []CustomHTTPConfigCase
	err = yaml.Unmarshal(b, &tests)
	require.NoError(t, err, "Unable to unmarshal tests to yaml.")

	for _, tcase := range tests {
		t.Run(tcase.Name, func(t *testing.T) {
			schHandler, errs := NewHandler(tcase.GQLSchema)
			require.NoError(t, errs)
			sch, err := FromString(schHandler.GQLSchema())
			require.NoError(t, err)

			var vars map[string]interface{}
			if tcase.GQLVariables != "" {
				err = json.Unmarshal([]byte(tcase.GQLVariables), &vars)
				require.NoError(t, err)
			}

			op, err := sch.Operation(
				&Request{
					Query:     tcase.GQLQuery,
					Variables: vars,
				})
			require.NoError(t, err)
			require.NotNil(t, op)

			var field Field
			if tcase.Type == "query" {
				queries := op.Queries()
				require.Len(t, queries, 1)
				field = queries[0]
			} else if tcase.Type == "mutation" {
				mutations := op.Mutations()
				require.Len(t, mutations, 1)
				field = mutations[0]
			} else if tcase.Type == "field" {
				queries := op.Queries()
				require.Len(t, queries, 1)
				q := queries[0]
				require.Len(t, q.SelectionSet(), 1)
				// We are allow checking the custom http config on the first field of the query.
				field = q.SelectionSet()[0]
			}

			c, err := field.CustomHTTPConfig()
			require.NoError(t, err)

			remoteSchemaHandler, errs := NewHandler(tcase.RemoteSchema)
			require.NoError(t, errs)
			remoteSchema, err := FromString(remoteSchemaHandler.GQLSchema())
			require.NoError(t, err)

			// Validate the generated query against the remote schema.
			tmpl, ok := (*c.Template).(map[string]interface{})
			require.True(t, ok)

			require.Equal(t, tcase.RemoteQuery, c.RemoteGqlQuery)

			v, _ := tmpl["variables"].(map[string]interface{})
			var rv map[string]interface{}
			if tcase.RemoteVariables != "" {
				require.NoError(t, json.Unmarshal([]byte(tcase.RemoteVariables), &rv))
			}
			require.Equal(t, rv, v)

			if tcase.InputVariables != "" {
				require.NoError(t, json.Unmarshal([]byte(tcase.InputVariables), &v))
			}
			op, err = remoteSchema.Operation(
				&Request{
					Query:     c.RemoteGqlQuery,
					Variables: v,
				})
			require.NoError(t, err)
			require.NotNil(t, op)
		})
	}
}

func TestAllowedHeadersList(t *testing.T) {
	// TODO Add Custom logic forward headers tests
	tcases := []struct {
		name      string
		schemaStr string
		expected  string
	}{
		{
			"auth header present in allowed headers list",
			`
	 type X @auth(
        query: {rule: """
          query {
            queryX(filter: { userRole: { eq: "ADMIN" } }) {
              __typename
            }
          }"""
        }
      ) {
        username: String! @id
        userRole: String @search(by: [hash])
	  }
	  # Dgraph.Authorization X-Test-Dgraph https://dgraph.io/jwt/claims RS256 "-----BEGIN PUBLIC KEY-----\nMIIBIjANBgkqhkiG9w0BAQEFAAOCAQ8AMIIBCgKCAQEAsppQMzPRyYP9KcIAg4CG\nUV3NGCIRdi2PqkFAWzlyo0mpZlHf5Hxzqb7KMaXBt8Yh+1fbi9jcBbB4CYgbvgV0\n7pAZY/HE4ET9LqnjeF2sjmYiGVxLARv8MHXpNLcw7NGcL0FgSX7+B2PB2WjBPnJY\ndvaJ5tsT+AuZbySaJNS1Ha77lW6gy/dmBDybZ1UU+ixRjDWEqPmtD71g2Fpk8fgr\nReNm2h/ZQsJ19onFaGPQN6L6uJR+hfYN0xmOdTC21rXRMUJT8Pw9Xsi6wSt+tI4T\nKxDfMTxKksfjv93dnnof5zJtIcMFQlSKLOrgDC0WP07gVTR2b85tFod80ykevvgu\nAQIDAQAB\n-----END PUBLIC KEY-----"
	`,
			"X-Test-Dgraph",
		},
	}
	for _, test := range tcases {
		t.Run(test.name, func(t *testing.T) {
			schHandler, errs := NewHandler(test.schemaStr)
			require.NoError(t, errs)
			_, err := FromString(schHandler.GQLSchema())
			require.NoError(t, err)
			require.True(t, strings.Contains(hc.allowed, test.expected))
		})
	}
}

func TestParseSecrets(t *testing.T) {
	tcases := []struct {
		name               string
		schemaStr          string
		expectedSecrets    map[string]string
		expectedAuthHeader string
		err                error
	}{
		{"should be able to parse secrets",
			`
			type User {
				id: ID!
				name: String!
			}

			 # Dgraph.Secret  GITHUB_API_TOKEN   "some-super-secret-token"
			# Dgraph.Secret STRIPE_API_KEY "stripe-api-key-value"
			`,
			map[string]string{"GITHUB_API_TOKEN": "some-super-secret-token",
				"STRIPE_API_KEY": "stripe-api-key-value"},
			"",
			nil,
		},
		{"should be able to parse secret where schema also has other comments.",
			`
		# Dgraph.Secret  GITHUB_API_TOKEN   "some-super-secret-token"

		type User {
			id: ID!
			name: String!
		}

		# Dgraph.Secret STRIPE_API_KEY "stripe-api-key-value"
		# random comment
		`,
			map[string]string{"GITHUB_API_TOKEN": "some-super-secret-token",
				"STRIPE_API_KEY": "stripe-api-key-value"},
			"",
			nil,
		},
		{
			"should throw an error if the secret is not in the correct format",
			`
			type User {
				id: ID!
				name: String!
			}

			# Dgraph.Secret RANDOM_TOKEN
			`,
			nil,
			"",
			errors.New("incorrect format for specifying Dgraph secret found for " +
				"comment: `# Dgraph.Secret RANDOM_TOKEN`, it should " +
				"be `# Dgraph.Secret key value`"),
		},
		{
			"should work along with authorization",
			`
			type User {
				id: ID!
				name: String!
			}

			# Dgraph.Secret  "GITHUB_API_TOKEN"   "some-super-secret-token"
			# Dgraph.Authorization X-Test-Dgraph https://dgraph.io/jwt/claims HS256 "key"
			# Dgraph.Secret STRIPE_API_KEY "stripe-api-key-value"
			`,
			map[string]string{"GITHUB_API_TOKEN": "some-super-secret-token",
				"STRIPE_API_KEY": "stripe-api-key-value"},
			"X-Test-Dgraph",
			nil,
		},
		{
			"should throw an error if multiple authorization values are specified",
			`
			type User {
				id: ID!
				name: String!
			}

			# Dgraph.Authorization random https://dgraph.io/jwt/claims HS256 "key"
			# Dgraph.Authorization X-Test-Dgraph https://dgraph.io/jwt/claims HS256 "key"
			`,
			nil,
			"",
			errors.New(`Dgraph.Authorization should be only be specified once in a schema` +
				`, found second mention: # Dgraph.Authorization X-Test-Dgraph` +
				` https://dgraph.io/jwt/claims HS256 "key"`),
		},
	}
	for _, test := range tcases {
		t.Run(test.name, func(t *testing.T) {
			s, err := parseSecrets(test.schemaStr)
			if test.err != nil || err != nil {
				require.EqualError(t, err, test.err.Error())
				return
			}

			require.Equal(t, test.expectedSecrets, s)
			if test.expectedAuthHeader != "" {
				require.Equal(t, test.expectedAuthHeader, authorization.GetHeader())
			}
		})
	}
}
