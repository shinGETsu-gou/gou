/*
 * Copyright (c) 2015, Shinya Yagyu
 * All rights reserved.
 * Redistribution and use in source and binary forms, with or without
 * modification, are permitted provided that the following conditions are met:
 *
 * 1. Redistributions of source code must retain the above copyright notice,
 *    this list of conditions and the following disclaimer.
 * 2. Redistributions in binary form must reproduce the above copyright notice,
 *    this list of conditions and the following disclaimer in the documentation
 *    and/or other materials provided with the distribution.
 * 3. Neither the name of the copyright holder nor the names of its
 *    contributors may be used to endorse or promote products derived from this
 *    software without specific prior written permission.
 *
 * THIS SOFTWARE IS PROVIDED BY THE COPYRIGHT HOLDERS AND CONTRIBUTORS "AS IS"
 * AND ANY EXPRESS OR IMPLIED WARRANTIES, INCLUDING, BUT NOT LIMITED TO, THE
 * IMPLIED WARRANTIES OF MERCHANTABILITY AND FITNESS FOR A PARTICULAR PURPOSE
 * ARE DISCLAIMED. IN NO EVENT SHALL THE COPYRIGHT HOLDER OR CONTRIBUTORS BE
 * LIABLE FOR ANY DIRECT, INDIRECT, INCIDENTAL, SPECIAL, EXEMPLARY, OR
 * CONSEQUENTIAL DAMAGES (INCLUDING, BUT NOT LIMITED TO, PROCUREMENT OF
 * SUBSTITUTE GOODS OR SERVICES; LOSS OF USE, DATA, OR PROFITS; OR BUSINESS
 * INTERRUPTION) HOWEVER CAUSED AND ON ANY THEORY OF LIABILITY, WHETHER IN
 * CONTRACT, STRICT LIABILITY, OR TORT (INCLUDING NEGLIGENCE OR OTHERWISE)
 * ARISING IN ANY WAY OUT OF THE USE OF THIS SOFTWARE, EVEN IF ADVISED OF THE
 * POSSIBILITY OF SUCH DAMAGE.
 */

package gou

import (
	"errors"
	"fmt"
	"html/template"
	"log"
	"math"
	"math/rand"
	"net/http"
	"os"
	"path"
	"regexp"
	"sort"
	"strings"
	"time"
)

type cgi struct {
	m          message
	host       string
	filter     string
	strFilter  string
	tag        string
	strTag     string
	remoteaddr string
	isAdmin    bool
	isFriend   bool
	isVisitor  bool
	jc         *jsCache
	req        *http.Request
	wr         http.ResponseWriter
	path       string
}

func newCGI(w http.ResponseWriter, r *http.Request) *cgi {
	c := &cgi{
		remoteaddr: r.RemoteAddr,
		jc:         newJsCache(absDocroot),
		wr:         w,
	}
	c.m = searchMessage(r.Header.Get("Accept-Language"))
	c.isAdmin = reAdmin.MatchString(c.remoteaddr)
	c.isFriend = reFriend.MatchString(c.remoteaddr)
	c.isVisitor = reVisitor.MatchString(c.remoteaddr)
	c.req = r
	err := r.ParseForm()
	if err != nil {
		log.Println(err)
		return nil
	}
	p := strings.Split(r.URL.Path, "/")
	if len(p) > 1 {
		c.path = strings.Join(p[1:], "/")
	}
	return c
}

func (c *cgi) extension(suffix string, useMerged bool) []string {
	var filename []string
	var merged string
	err := eachFiles(absDocroot, func(f os.FileInfo) error {
		i := f.Name()
		if strings.HasSuffix(i, "."+suffix) && (!strings.HasPrefix(i, ".") || strings.HasPrefix(i, "_")) {
			filename = append(filename, i)
		} else {
			if useMerged && i == "__merged."+suffix {
				merged = i
			}
		}
		return nil
	})
	if err != nil {
		log.Println(err)
	}

	if merged != "" {
		return []string{merged}
	}
	sort.Strings(filename)
	return filename
}

type Menubar struct {
	GatewayLink
	GatewayCGI string
	Message    message
	ID         string
	RSS        string
	IsAdmin    bool
	IsFriend   bool
}

func (c *cgi) makeMenubar(id, rss string) *Menubar {
	g := &Menubar{
		GatewayCGI: gatewayURL,
		Message:    c.m,
		ID:         id,
		RSS:        rss,
		IsAdmin:    c.isAdmin,
		IsFriend:   c.isFriend,
	}
	g.GatewayLink.Message = c.m
	return g
}

func (c *cgi) footer(menubar *Menubar) {
	g := struct {
		Menubar *Menubar
	}{
		menubar,
	}
	renderTemplate("footer", g, c.wr)
}

func (c *cgi) rfc822Time(stamp int64) string {
	return time.Unix(stamp, 0).Format("2006-01-02 15:04:05")
}

func (c *cgi) printParagraph(contents string) {
	g := struct {
		Contents string
	}{
		Contents: contents,
	}
	renderTemplate("paragraph", g, c.wr)
}

type Header struct {
	Message    message
	RootPath   string
	Title      string
	RSS        string
	DenyRobot  bool
	Mergedjs   *jsCache
	JS         []string
	CSS        []string
	Menu       Menubar
	Dummyquery int64
	ThreadCGI  string
	AppliType  string
}

func (c *cgi) header(title, rss string, cookie []*http.Cookie, denyRobot bool, menu *Menubar) {
	if rss == "" {
		rss = gatewayURL + "/rss"
	}
	var js []string
	if c.req.FormValue("__debug_js") != "" {
		js = c.extension("js", false)
	} else {
		c.jc.update()
	}
	h := &Header{
		c.m,
		rootPath,
		title,
		rss,
		denyRobot,
		c.jc,
		js,
		c.extension("css", false),
		*menu,
		time.Now().Unix(),
		threadURL,
		"thread",
	}
	if cookie != nil {
		for _, co := range cookie {
			http.SetCookie(c.wr, co)
		}
	}
	renderTemplate("header", h, c.wr)
}

func (c *cgi) resAnchor(id, appli string, title string, absuri bool) string {
	title = strEncode(title)
	var prefix, innerlink string
	if absuri {
		prefix = "http://" + c.host
	} else {
		innerlink = " class=\"innderlink\""
	}
	return fmt.Sprintf("<a href=\"%s%s%s%s/%s\"%s>", prefix, appli, querySeparator, title, id, innerlink)
}

func (c *cgi) htmlFormat(plain, appli string, title string, absuri bool) string {
	buf := strings.Replace(plain, "<br>", "\n", -1)
	buf = strings.Replace(buf, "\t", "        ", -1)
	buf = escape(buf)
	reg := regexp.MustCompile("https?://[^\\x00-\\x20\"'()<>\\[\\]\\x7F-\\xFF]{2,}")
	buf = reg.ReplaceAllString(buf, "<a href=\"\\g<0>\">\\g<0></a>")
	reg = regexp.MustCompile("(&gt;&gt;)([0-9a-f]{8})")
	id := reg.ReplaceAllString(buf, "\\2")
	buf = reg.ReplaceAllString(buf, c.resAnchor(id, appli, title, absuri)+"\\g<0></a>")

	var tmp string
	reg = regexp.MustCompile("\\[\\[([^<>]+?)\\]\\]")
	for buf != "" {
		if reg.MatchString(buf) {
			reg.ReplaceAllStringFunc(buf, func(str string) string {
				return c.bracketLink(str, appli, absuri)
			})
		} else {
			tmp += buf
			break
		}
	}
	return escapeSpace(tmp)
}

func (c *cgi) bracketLink(link, appli string, absuri bool) string {

	var prefix string
	if absuri {
		prefix = "http://" + c.host
	}
	reg := regexp.MustCompile("^/(thread)/([^/]+)/([0-9a-f]{8})$")
	if m := reg.FindStringSubmatch(link); m != nil {
		url := prefix + threadURL + querySeparator + strEncode(m[2]) + "/" + m[3]
		return "<a href=\"" + url + "\" class=\"reclink\">[[" + link + "]]</a>"
	}

	reg = regexp.MustCompile("^/(thread)/([^/]+)$")
	if m := reg.FindStringSubmatch(link); m != nil {
		uri := prefix + application[m[1]] + querySeparator + strEncode(m[2])
		return "<a href=\"" + uri + "\">[[" + link + "]]</a>"
	}

	reg = regexp.MustCompile("^([^/]+)/([0-9a-f]{8})$")
	if m := reg.FindStringSubmatch(link); m != nil {
		uri := prefix + appli + querySeparator + strEncode(m[1]) + "/" + m[2]
		return "<a href=\"" + uri + "\" class=\"reclink\">[[" + link + "]]</a>"
	}

	reg = regexp.MustCompile("^([^/]+)$")
	if m := reg.FindStringSubmatch(link); m != nil {
		uri := prefix + appli + querySeparator + strEncode(m[1])
		return "<a href=\"" + uri + "\">[[" + link + "]]</a>"
	}
	return "[[" + link + "]]"
}

func (c *cgi) removeFileForm(ca *cache, title string) {
	s := struct {
		Cache    *cache
		Title    string
		IsAdmin  bool
		AdminCGI string
		Message  message
	}{
		ca,
		title,
		c.isAdmin,
		adminURL,
		c.m,
	}
	renderTemplate("remove_file_form", s, c.wr)
}

func (c *cgi) mchURL() string {
	path := "/2ch/subject.txt"
	if !enable2ch {
		return ""
	}
	if serverName != "" {
		return "//" + serverName + path
	}
	reg := regexp.MustCompile(":\\d+")
	host := reg.ReplaceAllString(c.req.Host, "")
	if host == "" {
		return ""
	}
	return fmt.Sprintf("//%s:%d%s", host, datPort, path)
}

type mchCategory struct {
	URL  string
	Text string
}

func (c *cgi) mchCategories() []*mchCategory {
	var categories []*mchCategory
	if !enable2ch {
		return categories
	}
	mchURL := c.mchURL()
	err := eachLine(runDir+"/tag.txt", func(line string, i int) error {
		tag := strings.TrimRight(line, "\r\n")
		catURL := strings.Replace(mchURL, "2ch", fileEncode("2ch", tag), -1)
		categories = append(categories, &mchCategory{
			catURL,
			tag,
		})
		return nil
	})
	if err != nil {
		log.Println(err)
	}

	return categories
}

func (c *cgi) printJump(next string) {
	s := struct {
		Next template.HTML
	}{
		template.HTML(next),
	}
	renderTemplate("jump", s, c.wr)
}

func (c *cgi) print302(next string) {
	c.header("Loading...", "", nil, false, nil)
	c.printJump(next)
	c.footer(nil)
}
func (c *cgi) print403(next string) {
	c.header(c.m["403"], "", nil, true, nil)
	c.printParagraph(c.m["403_body"])
	c.printJump(next)
	c.footer(nil)
}
func (c *cgi) print404(ca *cache, id string) {
	c.header(c.m["404"], "", nil, true, nil)
	c.printParagraph(c.m["404_body"])
	if ca != nil {
		c.removeFileForm(ca, "")
	}
	c.footer(nil)
}
func touch(fname string) {
	f, err := os.Create(fname)
	defer close(f)
	if err != nil {
		log.Println(err)
	}
}

func (c *cgi) lock() bool {
	var lockfile string
	if c.isAdmin {
		lockfile = adminSearch
	} else {
		lockfile = searchLock
	}
	if !isFile(lockfile) {
		touch(lockfile)
		return true
	}
	s, err := os.Stat(lockfile)
	if err != nil {
		log.Println(err)
		return false
	}
	if s.ModTime().Add(searchTimeout).Before(time.Now()) {
		touch(lockfile)
		return true
	}
	return false
}

func (c *cgi) unlock() {
	var lockfile string
	if c.isAdmin {
		lockfile = adminSearch
	} else {
		lockfile = searchLock
	}
	err := os.Remove(lockfile)
	if err != nil {
		log.Println(err)
	}
}

func (c *cgi) getCache(ca *cache) bool {
	result := ca.search(nil)
	c.unlock()
	return result
}

func (c *cgi) printNewElementForm() {
	if !c.isAdmin && !c.isFriend {
		return
	}
	s := struct {
		Datfile    string
		CGIname    string
		Message    message
		TitleLimit int
		IsAdmin    bool
	}{
		"",
		gatewayURL,
		c.m,
		titleLimit,
		c.isAdmin,
	}
	renderTemplate("new_element_form", s, c.wr)
}

type attached struct {
	Filename string
	Data     []byte
}

func (c *cgi) parseAttached() (*attached, error) {
	err := c.req.ParseMultipartForm(int64(recordLimit) << 10)
	if err != nil {
		return nil, err
	}
	attach := c.req.MultipartForm
	if len(attach.File) > 0 {
		filename := attach.Value["filename"][0]
		fpStrAttach := attach.File[filename][0]
		f, err := fpStrAttach.Open()
		defer close(f)
		if err != nil {
			return nil, err
		}
		var strAttach = make([]byte, recordLimit<<10)
		_, err = f.Read(strAttach)
		if err == nil || err.Error() != "EOF" {
			c.header(c.m["big_file"], "", nil, true, nil)
			c.footer(nil)
			return nil, err
		}
		return &attached{
			filename,
			strAttach,
		}, nil
	}
	return nil, errors.New("attached file not found")
}

//errorTime calculates gaussian distribution by box-muller transformation.
func (c *cgi) errorTime() int64 {
	x1 := rand.Float64()
	x2 := rand.Float64()
	return int64(timeErrorSigma*math.Sqrt(-2*math.Log(x1))*math.Cos(2*math.Pi*x2)) + time.Now().Unix()
}

func (c *cgi) doPost() string {
	attached, attachedErr := c.parseAttached()
	if attachedErr != nil {
		log.Println(attachedErr)
	}
	guessSuffix := "txt"
	if attachedErr == nil {
		e := path.Ext(attached.Filename)
		if e != "" {
			guessSuffix = strings.ToLower(e)
		}
	}

	suffix := c.req.FormValue("suffix")
	switch {
	case suffix == "" || suffix == "AUTO":
		suffix = guessSuffix
	case strings.HasPrefix(suffix, "."):
		suffix = suffix[1:]
	}
	suffix = strings.ToLower(suffix)
	reg := regexp.MustCompile("[^0-9A-Za-z]")
	suffix = reg.ReplaceAllString(suffix, "")

	stamp := time.Now().Unix()
	if c.req.FormValue("error") != "" {
		stamp = c.errorTime()
	}

	ca := newCache(c.req.FormValue("file"))
	body := make(map[string]string)
	if value := c.req.FormValue("body"); value != "" {
		body["body"] = escape(value)
	}

	if attachedErr == nil {
		body["attach"] = string(attached.Data)
		body["suffix"] = strings.TrimSpace(suffix)
	}
	if len(body) == 0 {
		c.header(c.m["null_article"], "", nil, true, nil)
		c.footer(nil)
		return ""
	}
	rec := newRecord(ca.Datfile, "")
	passwd := c.req.FormValue("passwd")
	id := rec.build(stamp, body, passwd)

	proxyClient := c.req.Header.Get("X_FORWARDED_FOR")
	log.Printf("post %s/%d_%s from %s/%s\n", ca.Datfile, stamp, id, c.remoteaddr, proxyClient)

	if len(rec.recstr()) > recordLimit<<10 {
		c.header(c.m["big_file"], "", nil, true, nil)
		c.footer(nil)
		return ""
	}
	if spamCheck(rec.recstr()) {
		c.header(c.m["spam"], "", nil, true, nil)
		c.footer(nil)
		return ""
	}

	if ca.Exists() {
		ca.addData(rec)
		ca.syncStatus()
	} else {
		c.print404(nil, "")
		return ""
	}

	if c.req.FormValue("dopost") != "" {
		queue.append(rec, nil)
		go queue.run()
	}

	return id[:8]

}

func (c *cgi) printIndexList(cl *cacheList, target string, footer bool, searchNewFile bool) {
	s := struct {
		Target        string
		Filter        string
		Tag           string
		Taglist       *UserTagList
		Chachelist    *cacheList
		GatewayCGI    string
		AdminCGI      string
		Message       message
		SearchNewFile bool
		IsAdmin       bool
		IsFriend      bool
		Types         []string
		GatewayLink
		ListItem
	}{
		target,
		c.strFilter,
		c.strTag,
		userTagList,
		cl,
		gatewayURL,
		adminURL,
		c.m,
		searchNewFile,
		c.isAdmin,
		c.isFriend,
		types,
		GatewayLink{
			Message: c.m,
		},
		ListItem{
			IsAdmin: c.isAdmin,
			filter:  c.filter,
			tag:     c.tag,
		},
	}
	renderTemplate("index_list", s, c.wr)
	if footer {
		c.printNewElementForm()
		c.footer(nil)
	}
}

func (c *cgi) checkGetCache() bool {
	if !c.isAdmin && !c.isFriend {
		return false
	}
	reg, err := regexp.Compile(robot)
	if err != nil {
		log.Print(err)
		return false
	}
	if reg.MatchString(c.req.Header.Get("User-Agent")) {
		return false
	}
	if c.lock() {
		return true
	}
	return false
}

func (c *cgi) checkVisitor() bool {
	return c.isAdmin || c.isFriend || c.isVisitor
}
