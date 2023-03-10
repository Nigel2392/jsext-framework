package router

import (
	"strings"

	"github.com/Nigel2392/jsext"
	"github.com/Nigel2392/jsext/elements"
)

// A url map element
type URLs struct {
	order []string
	urls  map[string]*elements.Element
	// mu    *sync.Mutex
}

// Create a new url map
func NewURLs() *URLs {
	return &URLs{
		urls:  make(map[string]*elements.Element),
		order: make([]string, 0),
		// mu:    &sync.Mutex{},
	}
}

// Get a url element from the map
func (u *URLs) Get(key string) *elements.Element {
	key = strings.ToUpper(key)
	return u.urls[key]
}

// Set a url element in the map
func (u *URLs) Set(key string, value *elements.Element, external ...bool) {
	if len(external) == 0 || len(external) > 0 && !external[0] {
		var href = value.GetAttr("href")
		// if len(href) > 6 && href[:6] != router.RT_PREFIX {
		value.Delete("href")
		value.AttrHref(RT_PREFIX + href)
		// }
	}
	if value.Text == "" {
		value.Text = key
	}
	key = strings.ToUpper(key)
	if _, ok := u.urls[key]; ok {
		panic("URL already exists: " + key)
	}
	// u.mu.Lock()
	u.urls[key] = value
	u.order = append(u.order, key)
	// u.mu.Unlock()
}

func (u *URLs) SetRaw(key string, value *elements.Element) {
	key = strings.ToUpper(key)
	if _, ok := u.urls[key]; ok {
		panic("URL already exists: " + key)
	}
	// u.mu.Lock()
	u.urls[key] = value
	u.order = append(u.order, key)
	// u.mu.Unlock()
}

// Delete a url element from the map, and remove it from the DOM
func (u *URLs) Delete(key string) {
	var ok bool
	var element *elements.Element
	key = strings.ToUpper(key)
	if element, ok = u.urls[key]; !ok {
		panic("URL does not exist: " + key)
	}
	// u.mu.Lock()
	delete(u.urls, key)
	element.Remove()
	for i, v := range u.order {
		if v == key {
			u.order = append(u.order[:i], u.order[i+1:]...)
			break
		}
	}
	// u.mu.Unlock()
}

// Set display to none for all urls,
// or for a list of urls
func (u *URLs) Hide(urlname ...string) {
	if len(urlname) == 0 {
		for _, v := range u.urls {
			v.AttrStyle("display:none")
		}
		return
	}
	for _, v := range urlname {
		var url = u.Get(v)
		if url != nil {
			url.AttrStyle("display:none")
		}
	}
}

// Set display to param_display for all urls,
// or for a list of urls
func (u *URLs) Show(display string, urlname ...string) {
	if len(urlname) == 0 {
		for _, v := range u.urls {
			v.AttrStyle("display:" + display)
		}
		return
	}
	for _, v := range urlname {
		var url = u.Get(v)
		if url != nil {
			url.AttrStyle("display:" + display)
		}
	}
}

// Loop through all urls in order
func (u *URLs) InOrder(reverse ...bool) []*elements.Element {
	var ret = make([]*elements.Element, 0)
	if len(reverse) > 0 && reverse[0] {
		for i := len(u.order) - 1; i >= 0; i-- {
			ret = append(ret, u.urls[u.order[i]])
		}
		return ret
	}
	for _, v := range u.order {
		ret = append(ret, u.urls[v])
	}
	return ret
}

// Loop through all key, value urls in order
func (u *URLs) ForEach(f func(k string, elem *elements.Element), reverse ...bool) {
	if len(reverse) > 0 && reverse[0] {
		for i := len(u.order) - 1; i >= 0; i-- {
			f(u.order[i], u.urls[u.order[i]])
		}
		return
	}
	for _, orderedKey := range u.order {
		f(orderedKey, u.urls[orderedKey])
	}
}

// Get the underlying map of URLs
func (u *URLs) Map() map[string]*elements.Element {
	return u.urls
}

// Length of the underlying map of URLs
func (u *URLs) Len() int {
	return len(u.urls)
}

// Get the underlying slice of ordered keys
func (u *URLs) Keys() []string {
	return u.order
}

// Fill up the URLs map from a slice of Elements
func (u *URLs) FromElements(raw bool, elems ...*elements.Element) {
	if raw {
		for _, v := range elems {
			u.SetRaw(v.Text, v)
		}
		return
	}
	for _, v := range elems {
		var href = v.GetAttr("href")
		if strings.HasPrefix(href, RT_PREFIX_EXTERNAL) {
			v.Delete("href")
			v.AttrHref(strings.TrimPrefix(href, RT_PREFIX_EXTERNAL))
			u.Set(v.Text, v, true)
			continue
		}
		u.Set(v.Text, v)
	}
}

// Generate the urls from a slice of maps
// Each map has the following attributes:
//   - (string) name: the name of the url
//   - (string) href: the href of the url
//   - (string) text: the text of the url
//   - (bool) external: whether the url is external or not
//   - (bool) Hide: whether the url is hidden or not
//   - ([]string) class: the classes of the url (optional)
//   - (string) id: the id of the url (optional)
//   - ([]string) style: the style of the url (optional)
func (u *URLs) FromMap(maps ...map[string]interface{}) {
	for _, v := range maps {
		var name, href, text string
		var external, hide bool
		var classes, styles []string
		var id string
		if v["name"] != nil {
			name = v["name"].(string)
		}
		if v["href"] != nil {
			href = v["href"].(string)
		}
		if v["text"] != nil {
			text = v["text"].(string)
		}
		if v["external"] != nil {
			external = v["external"].(bool)
		}
		if v["hide"] != nil {
			hide = v["hide"].(bool)
		}
		if v["class"] != nil {
			classes = v["classes"].([]string)
		}
		if v["id"] != nil {
			id = v["id"].(string)
		}
		if v["style"] != nil {
			styles = v["style"].([]string)
		}
		var elem = elements.A(href, text)
		if hide {
			elem.AttrStyle("display:none")
		}
		if len(classes) > 0 {
			elem.AttrClass(classes...)
		}
		if id != "" {
			elem.SetAttr("id", id)
		}
		if len(styles) > 0 {
			elem.AttrStyle(styles...)
		}
		u.Set(name, elem, external)
	}
}

// On click action for URLs
// Takes a function that takes a URL element and the event.target | this
func (u *URLs) OnClick(f func(*elements.Element, jsext.Value)) {
	for _, v := range u.urls {
		v.AddEventListener("click", func(this jsext.Value, event jsext.Event) {
			var target = event.Target()
			if target.IsUndefined() || target.IsNull() {
				target = this
			}
			f(v, target)
		})
	}
}
