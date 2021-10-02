// Copyright ©2021 Dan Kortschak. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// The xcompose command generates a DefaultKeyBindings.dict key binding map from an
// X11 Compose definition file.
package main

//go:generate go run generate.go
//go:generate go fmt .

import (
	"bufio"
	"bytes"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"sort"
	"strconv"
	"strings"

	// For Compose.pre.
	_ "embed"
)

// File obtained from https://cgit.freedesktop.org/xorg/lib/libX11/plain/nls/en_US.UTF-8/Compose.pre
//go:embed Compose.pre
var compose string

func main() {
	var (
		dump  = flag.Bool("dump", false, "dump the xcompose config file to output.")
		out   = flag.String("o", "", "output destination — stdout if empty.")
		altGr = flag.String("altgr", "§", "rune to bind AltGr to (use Karabiner-Elements).")
		help  = flag.Bool("help", false, "display help.")
	)

	flag.Parse()
	flag.Usage = func() {
		fmt.Fprintf(flag.CommandLine.Output(), `Usage of %s: 

xcompose generates a DefaultKeyBindings.dict key binding map from an X11 Compose
definition file. Using the dict file depends on mapping a sensible modifier key
to a character. This can be done with Karabiner-Elements. By default the AltGr
key is mapped to '§'.

The generated dictionary is then placed in ~/Library/KeyBindings/DefaultKeyBindings.dict.

`, os.Args[0])
		flag.PrintDefaults()
	}
	if *help {
		flag.Usage()
		os.Exit(0)
	}

	w := os.Stdout
	if *out != "" {
		var err error
		w, err = os.Create(*out)
		if err != nil {
			log.Fatal(err)
		}
		defer w.Close()
	}

	if *dump {
		err := dumpCompose(w)
		if err != nil {
			log.Fatal(err)
		}
	}

	runes := []rune(*altGr)
	if len(runes) != 1 {
		flag.Usage()
		os.Exit(2)
	}
	user := map[string]rune{"<Multi_key>": runes[0]}

	err := format(w, buildDict(user, nil), 0)
	if err != nil {
		log.Fatal(err)
	}
}

func dumpCompose(w io.Writer) error {
	_, err := io.Copy(w, strings.NewReader(compose))
	return err
}

func buildDict(user map[string]rune, src io.Reader) map[string]interface{} {
	mapping := make(map[string]interface{})
	if src == nil {
		src = strings.NewReader(compose)
	}
	sc := bufio.NewScanner(src)
	for sc.Scan() {
		if !bytes.HasPrefix(sc.Bytes(), []byte{'<'}) {
			continue
		}
		parts := strings.FieldsFunc(sc.Text(), func(r rune) bool {
			return r == ':'
		})
		if len(parts) < 2 {
			log.Fatalf("unexpected number of parts: %s", sc.Text())
		}
		path := strings.Fields(parts[0])
		for i, p := range path {
			k, err := keyFor(p, user)
			if err != nil {
				continue
			}
			path[i] = k
		}
		val, err := strconv.Unquote(strings.FieldsFunc(parts[1], func(r rune) bool {
			return r == ' ' || r == '\t'
		})[0])
		if err != nil {
			log.Fatalf("failed to unquote value %s: %v", sc.Text(), err)
		}
		known := true
		for _, p := range path {
			if strings.HasPrefix(p, "<") && strings.HasSuffix(p, ">") {
				known = false
				break
			}
		}
		if !known {
			continue
		}
		insert(mapping, val, path...)
	}
	return mapping
}

func keyFor(name string, user map[string]rune) (string, error) {
	if strings.HasPrefix(name, "<U") {
		utf, err := strconv.ParseInt(strings.TrimSuffix(strings.TrimPrefix(name, "<U"), ">"), 16, 32)
		if err != nil {
			return "", err
		}
		return string(rune(utf)), nil
	}
	val, ok := keysymdef[name]
	if ok {
		return string(val), nil
	}
	val, ok = user[name]
	if ok {
		return string(val), nil
	}
	return "", fmt.Errorf("no value for %s", name)
}

func insert(dst map[string]interface{}, val string, path ...string) {
	if len(path) == 0 {
		return
	}
	if dst == nil {
		dst = make(map[string]interface{})
	}
	if len(path) == 1 {
		dst[path[0]] = val
		return
	}
	child, ok := dst[path[0]]
	if !ok {
		child = make(map[string]interface{})
		dst[path[0]] = child
	}
	dst, ok = child.(map[string]interface{})
	if !ok {
		return
	}
	insert(dst, val, path[1:]...)
}

func format(w io.Writer, dict map[string]interface{}, depth int) error {
	_, err := fmt.Fprintln(w, "{")
	if err != nil {
		return err
	}
	keys := make([]string, 0, len(dict))
	for k := range dict {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		_, err = fmt.Fprintf(w, "%s%s = ", strings.Repeat("\t", depth+1), quote(k))
		if err != nil {
			return err
		}
		switch val := dict[k].(type) {
		case string:
			_, err = fmt.Fprintf(w, "(\"insertText:\", %s);\n", quote(val))
			if err != nil {
				return err
			}
		case map[string]interface{}:
			err = format(w, val, depth+1)
			if err != nil {
				return err
			}
		}
	}
	_, err = fmt.Fprintf(w, "%s}", strings.Repeat("\t", depth))
	if err != nil {
		return err
	}
	if depth != 0 {
		_, err = fmt.Fprint(w, ";")
		if err != nil {
			return err
		}
	}
	_, err = fmt.Fprintln(w)
	return err
}

func quote(s string) string {
	s = strconv.Quote(s)
	if strings.HasPrefix(s, `"\u`) {
		s = strings.ToUpper(s)
	}
	return s
}
