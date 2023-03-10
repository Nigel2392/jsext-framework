package scroll

import (
	"strconv"
	"strings"
	"syscall/js"
	"time"
	"unicode"

	"github.com/Nigel2392/jsext"
	"github.com/Nigel2392/jsext-framework/components"
	"github.com/Nigel2392/jsext-framework/helpers"
	"github.com/Nigel2392/jsext/requester"
)

type PageDirection int8

const (
	Initial PageDirection = 0
	Up      PageDirection = 1
	Down    PageDirection = 2
	Left    PageDirection = 3
	Right   PageDirection = 4
)

// Axis to scroll on.
type Axis int8

// Axis to scroll on.
const (
	ScrollAxisX Axis = 1
	ScrollAxisY Axis = 2
)

// Waiter to lock the main thread.
var waiter = make(chan struct{})

// Page struct.
// Mostly used internally.
type Page struct {
	title     string
	hash      string
	Component components.ComponentWithValue
	OnShow    func(element components.ComponentWithValue, p PageDirection)
	OnHide    func(element components.ComponentWithValue, p PageDirection)
}

// Callback for when the page is being viewed.
func (p *Page) SetOnShow(cb func(element components.ComponentWithValue, p PageDirection)) *Page {
	p.OnShow = cb
	return p
}

// Callback for when the page is being hidden.
func (p *Page) SetOnHide(cb func(element components.ComponentWithValue, p PageDirection)) *Page {
	p.OnHide = cb
	return p
}

// Application options
type Options struct {
	ScrollAxis    Axis
	ClassPrefix   string
	ScrollThrough bool
}

func (o *Options) setDefaults() {
	if o.ClassPrefix == "" {
		o.ClassPrefix = "jsext-scrollable-app"
	}
	if o.ScrollAxis == 0 {
		o.ScrollAxis = ScrollAxisY
	}
}

// Application struct
type Application struct {
	Loader         components.Loader
	navbar         components.Component
	footer         components.Component
	pages          []*Page
	documentObject jsext.Element
	backgrounds    []*Background
	onPageChange   func(index int)
	currentPage    int
	Options        *Options
	clientFunc     func() *requester.APIClient
	client         *requester.APIClient
}

// Create a new application from options
func App(documentObjectQuerySelector string, options *Options) *Application {
	var object, err = jsext.QuerySelector(documentObjectQuerySelector)
	if err != nil {
		object = jsext.Body
	}
	var s = &Application{
		pages:          make([]*Page, 0),
		documentObject: object,
		Options:        options,
	}
	return s
}

// Set the application loader
func (s *Application) SetLoader(loader components.Loader) {
	s.Loader = loader
}

// Set the application navbar
func (s *Application) SetNavbar(c components.Component) {
	s.navbar = c
}

// Set the application footer
func (s *Application) SetFooter(c components.Component) {
	s.footer = c
}

// Set the application backgrounds
func (s *Application) Backgrounds(t BackgroundType, b ...string) Backgrounds {
	var backgrounds = make([]*Background, len(b))
	for i, background := range b {
		backgrounds[i] = &Background{
			BackgroundType: t,
			Background:     background,
			Gradient: &Gradient{
				Gradients: make([]string, 0),
			},
		}
	}
	s.backgrounds = append(s.backgrounds, backgrounds...)
	return backgrounds
}

// Add a page
func (s *Application) AddPage(title string, c components.ComponentWithValue) *Page {
	var page = &Page{
		title:     title,
		hash:      makeSlug(title),
		Component: c,
	}
	s.pages = append(s.pages, page)
	return page
}

// Run the application
func (s *Application) Run() {

	s.Options.setDefaults()

	// Render navbar
	if s.navbar != nil {
		var navbar = s.navbar.Render()
		navbar.ClassList().Add(s.Options.ClassPrefix + "-navbar")
		s.documentObject.AppendChild(navbar)
	}
	// Append the CSS to the document
	var displayDirection string
	var overflowAxis, oppositeOverflowAxis string
	var width string = `calc(100vw * ` + strconv.Itoa(len(s.pages)) + `);`
	var height string = `calc(100vh * ` + strconv.Itoa(len(s.pages)) + `);`
	var axis string
	switch s.Options.ScrollAxis {
	case ScrollAxisX:
		displayDirection = "row"
		overflowAxis = "overflow-x"
		oppositeOverflowAxis = "overflow-y"
		height = "100%"
		axis = "x"
	default:
		displayDirection = "column"
		overflowAxis = "overflow-y"
		oppositeOverflowAxis = "overflow-x"
		width = "100%"
		axis = "y"
	}
	// Styling
	jsext.StyleBlock(s.Options.ClassPrefix+"-navbar-css", func() string {
		var css = `
			* {
				margin: 0;
				padding: 0;
			}
			body {
				height: 100vh;
				width: 100vw;
				overflow: hidden;
			}
			.` + s.Options.ClassPrefix + `-scrollable-page-container {
				width: 100vw;
				height: 100vh;
				` + overflowAxis + `: hidden;
				scroll-behavior: smooth;
				scroll-snap-type: ` + axis + ` mandatory; 
				scroll-snap-stop: always;
				` + oppositeOverflowAxis + `: hidden;
			}
			.` + s.Options.ClassPrefix + `-scrollable-page {
				width: ` + width + `;
				height: ` + height + `;
				display: flex;
				flex-direction: ` + displayDirection + `;
			}
			.` + s.Options.ClassPrefix + `-page {
				scroll-snap-align: center;
				display: flex;
				flex-direction: column;
				align-items: center;
				justify-content: center;
				width: 100vw;
				height: 100vh;
				font-size: 1.5em;
			}
			`
		if s.navbar != nil {
			css += `.` + s.Options.ClassPrefix + `-navbar {
				position: fixed;
				top: 0;
				left: 0;
				right: 0;
				z-index: 1000;
			}`
		}
		if s.footer != nil {
			css += `.` + s.Options.ClassPrefix + `-footer {
				position: fixed;
				bottom: 0;
				left: 0;
				right: 0;
				z-index: 1000;
			}`
		}

		if len(s.backgrounds) > 0 {
			var ct int
			var bg *Background
			var backup = s.backgrounds[0]
			for _, page := range s.pages {
				bg, ct = helpers.GetColor(s.backgrounds, ct, backup)
				css += bg.CSS(`#` + page.hash)
			}
		} else {
			css += (&Background{
				BackgroundType: BackgroundTypeColor,
				Background:     "#333333",
			}).CSS(`.` + s.Options.ClassPrefix + `-page`)
		}

		return css
	}())
	// Create the application elements
	var scrollablePageContainer = jsext.CreateElement("section")
	scrollablePageContainer.ClassList().Add(s.Options.ClassPrefix + "-scrollable-page-container")
	var scrollablePage = jsext.CreateElement("section")
	scrollablePage.ClassList().Add(s.Options.ClassPrefix + "-scrollable-page")
	for _, page := range s.pages {
		var section = jsext.CreateElement("section")
		section.ClassList().Add(s.Options.ClassPrefix + "-page")
		section.Set("id", page.hash)
		var p = page.Component.Render()
		p.ClassList().Add(s.Options.ClassPrefix + "-page-content")
		section.AppendChild(p)
		scrollablePage.AppendChild(section)
	}
	scrollablePageContainer.AppendChild(scrollablePage)
	s.documentObject.AppendChild(scrollablePageContainer)

	// Render the footer
	if s.footer != nil {
		var footer = s.footer.Render()
		footer.ClassList().Add(s.Options.ClassPrefix + "-footer")
		s.documentObject.AppendChild(footer)
	}

	// Add the application eventlistener for the arrow keys
	jsext.Document.Call("addEventListener", "keydown", js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		if args[0].Get("key").String() == "ArrowRight" || args[0].Get("key").String() == "ArrowDown" {
			s.NextPage()
		} else if args[0].Get("key").String() == "ArrowLeft" || args[0].Get("key").String() == "ArrowUp" {
			s.PreviousPage()
		}
		return nil
	}))

	// Add the application eventlistener for the mouse wheel
	var scrolled = false
	jsext.Document.Call("addEventListener", "wheel", js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		var dy = args[0].Get("deltaY").Float()
		args[0].Call("preventDefault")
		if !scrolled {
			if dy > 10 {
				s.NextPage()
			} else if dy < -10 {
				s.PreviousPage()
			}
			scrolled = true
			go func() {
				time.Sleep(500 * time.Millisecond)
				scrolled = false
			}()
		}
		return nil
	}), jsext.MapToObject(map[string]any{"passive": false}).Value())

	jsext.Element(jsext.Window).AddEventListener("hashchange", func(this jsext.Value, event jsext.Event) {
		var hash = jsext.Window.Get("location").Get("hash").String()
		var page, index = s.PageByHash(strings.Split(hash, "#")[1])
		if page != nil {
			var direction PageDirection
			if index > s.currentPage {
				switch s.Options.ScrollAxis {
				case ScrollAxisX:
					direction = Right
				case ScrollAxisY:
					direction = Down
				}
			} else if index < s.currentPage {
				switch s.Options.ScrollAxis {
				case ScrollAxisX:
					direction = Left
				case ScrollAxisY:
					direction = Up
				}
			}
			var oldPage = s.pages[s.currentPage]
			if oldPage.OnHide != nil {
				oldPage.OnHide(oldPage.Component, direction)
			}
			s.currentPage = index
			if page.OnShow != nil {
				page.OnShow(page.Component, direction)
			}
		}
	})

	// Remove the preloader
	const JSEXT_PRELOADER_ID = "jsext-preload-container"
	if preloader, err := jsext.QuerySelector("#" + JSEXT_PRELOADER_ID); err == nil {
		preloader.Remove()
	}

	// Set the initial page
	var hash = jsext.Document.Get("location").Get("hash").String()
	if hash != "" {
		var page, index = s.PageByHash(hash[1:])
		if page.OnShow != nil {
			s.currentPage = index
			jsext.Document.Call("getElementById", page.hash).Call("scrollIntoView", jsext.MapToObject(map[string]any{
				"behavior": "smooth",
			}).Value())
			page.OnShow(page.Component, Initial)
		}
	}

	<-waiter
}

// Get the page's container.
func (s *Application) containerByIndex(index int) jsext.Element {
	return s.documentObject.Value().QuerySelectorAll("." + s.Options.ClassPrefix + "-page")[index]
}

// Exit the application.
func (s *Application) Close() {
	close(waiter)
}

// Go to the next page
func (s *Application) NextPage() {
	var p PageDirection
	switch s.Options.ScrollAxis {
	case ScrollAxisX:
		p = Right
	case ScrollAxisY:
		p = Down
	}
	s.updatePage(p)
}

// Go to the previous page
func (s *Application) PreviousPage() {
	var p PageDirection
	switch s.Options.ScrollAxis {
	case ScrollAxisX:
		p = Left
	case ScrollAxisY:
		p = Up
	}
	s.updatePage(p)
}

// Update the page
func (s *Application) updatePage(p PageDirection) {
	var lastPage = s.currentPage
	switch p {
	case Down, Right:
		s.currentPage++
	case Up, Left:
		s.currentPage--
	case Initial:
		s.currentPage = 0
	}
	if s.Options.ScrollThrough {
		if s.currentPage >= len(s.pages) {
			s.currentPage = 0
		} else if s.currentPage < 0 {
			s.currentPage = len(s.pages) - 1
		}
	} else {
		if s.currentPage >= len(s.pages) {
			s.currentPage = len(s.pages) - 1
		} else if s.currentPage < 0 {
			s.currentPage = 0
		}
	}
	if lastPage != s.currentPage {
		var currentPage = s.pages[lastPage]
		if currentPage.OnHide != nil {
			currentPage.OnHide(currentPage.Component, p)
		}
		var page = s.pages[s.currentPage]
		jsext.Document.Set("title", page.title)
		// always push state to /#hash
		js.Global().Get("history").Call("pushState", nil, nil, "#"+page.hash)
		s.containerByIndex(s.currentPage).ScrollIntoView(true)
		if page.OnShow != nil {
			page.OnShow(page.Component, p)
		}
		if s.onPageChange != nil {
			s.onPageChange(s.currentPage)
		}
	}
}

// Initialize a http client with a loader for a new request.
func (a *Application) Client() *requester.APIClient {
	if a.clientFunc != nil {
		a.client = a.clientFunc()
	} else {
		a.client = requester.NewAPIClient()
	}
	a.client.Before(a.Loader.Show)
	a.client.After(func() {
		a.Loader.Finalize()
		a.client = nil
	})
	return a.client
}

// Set the client function.
func (a *Application) SetClientFunc(f func() *requester.APIClient) *Application {
	a.clientFunc = f
	return a
}

func makeSlug(s string) string {
	s = strings.ToLower(s)
	var b strings.Builder
	lastLetter := false
	for _, r := range s {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			b.WriteRune(r)
			lastLetter = true
		} else {
			if lastLetter {
				b.WriteRune('-')
			}
			lastLetter = false
		}
	}
	return b.String()
}

// containerByName returns the page's container by name.
func (s *Application) ContainerByName(name string) jsext.Element {
	for i, page := range s.pages {
		if page.title == name {
			return s.containerByIndex(i)
		}
	}
	return jsext.Undefined().ToElement()
}

// PageByHash returns the page by the hash value.
func (s *Application) PageByHash(name string) (*Page, int) {
	for i, page := range s.pages {
		if page.hash == name {
			return page, i
		}
	}
	return nil, 0
}

// PageByTitle returns the page by the title.
func (s *Application) PageByTitle(name string) (*Page, int) {
	for i, page := range s.pages {
		if page.title == name {
			return page, i
		}
	}
	return nil, 0
}

func (s *Application) CurrentPage() *Page {
	return s.pages[s.currentPage]
}
