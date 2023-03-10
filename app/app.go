//go:build js && wasm
// +build js,wasm

package app

import (
	"net/url"
	"syscall/js"

	"github.com/Nigel2392/jsext"
	"github.com/Nigel2392/jsext-framework/components"
	"github.com/Nigel2392/jsext-framework/components/loaders"
	"github.com/Nigel2392/jsext-framework/router"
	"github.com/Nigel2392/jsext-framework/router/routes"
	"github.com/Nigel2392/jsext-framework/router/vars"
	"github.com/Nigel2392/jsext/elements"
	"github.com/Nigel2392/jsext/requester"
)

// Preloader to be removed. This should happen automatically from the JS-script.
const JSEXT_PRELOADER_ID = "jsext-preload-container"

// App export to be used for embedding other exports.
var AppExport jsext.Export

// Set application exports.
// Available in javascript console under:
//
//	jsext.App.((defined_methods))
func init() {
	AppExport = jsext.NewExport()
	AppExport.SetFunc("Exit", Exit)
	AppExport.RegisterToExport("App", jsext.JSExt)
}

// Waiter to lock the main thread.
var WAITER = make(chan struct{})

// Main application, holds router and is the core of the
type Application struct {
	BaseElemSelector string
	Router           components.Router
	client           *requester.APIClient
	Navbar           components.Component
	Footer           components.Component
	Loader           components.Loader
	Base             jsext.Element
	clientFunc       func() *requester.APIClient
	onErr            func(err error)
	onLoad           func()
	beforeLoad       func()
	Data             DataMap
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

// Initialize a new application.
// If id is empty, the application will be initialized on the body.
func App(querySelector string, rt ...components.Router) *Application {
	// Get the application body
	var elem jsext.Element
	var err error
	if querySelector == "" {
		elem = jsext.Body
	} else {
		elem, err = jsext.QuerySelector(querySelector)
		if err != nil {
			panic(err)
		}
	}
	// Get the application router
	var r components.Router
	if len(rt) > 0 {
		r = rt[0]
	} else {
		r = router.NewRouter()
		r.SkipTrailingSlash()
		r.NameToTitle(true)
	}
	// Return the application
	var a = &Application{
		BaseElemSelector: querySelector,
		Router:           r,
		Base:             elem,
		Loader:           loaders.NewLoader(querySelector, loaders.ID_LOADER, true, loaders.LoaderRing),
		Data:             make(map[string]interface{}),
	}
	return a
}

// Decide what happens on errors.
func (a *Application) OnError(f func(*Application, error)) {
	var newF = func(err error) {
		f(a, err)
	}
	a.onErr = newF
}

// Set the base navbar.
func (a *Application) SetNavbar(navbar components.Component) *Application {
	a.Navbar = navbar
	return a
}

// Set the base footer.
func (a *Application) SetFooter(footer components.Component) *Application {
	a.Footer = footer
	return a
}

// Set the base loader.
func (a *Application) SetLoader(loader components.Loader) *Application {
	a.Loader = loader
	return a
}

// Set the base style.
func (a *Application) SetStyle(style string) *Application {
	a.Base.SetAttribute("style", style)
	return a
}

// Set classes on the base element.
func (a *Application) SetClasses(class string) *Application {
	a.Base.SetAttribute("class", class)
	return a
}

// Set title on the document
func (a *Application) SetTitle(title string) *Application {
	jsext.Document.Set("title", title)
	return a
}

// Run the application.
func (a *Application) Run() int {
	return a.run()
}

// Function to be while the application is loading.
func (a *Application) OnLoad(f func()) *Application {
	a.onLoad = f
	return a
}

// Function to be ran before the application is loaded.
func (a *Application) BeforeLoad(f func()) *Application {
	a.beforeLoad = f
	return a
}

// Function to be ran before the router is loaded.
func (a *Application) OnRouterLoad(f func()) *Application {
	a.Router.OnLoad(f)
	return a
}

// Function to be ran before the page is rendered.
func (a *Application) OnPageChange(f func(*Application, vars.Vars, *url.URL)) *Application {
	var newF = func(v vars.Vars, u *url.URL) {
		f(a, v, u)
	}
	a.Router.OnPageChange(newF)
	return a
}

// Function to be ran after the page is rendered.
func (a *Application) AfterPageChange(f func(*Application, vars.Vars, *url.URL)) *Application {
	var newF = func(v vars.Vars, u *url.URL) {
		f(a, v, u)
	}
	a.Router.AfterPageChange(newF)
	return a
}

// Setup application to be ran.
// Return 0 on exit.
func (a *Application) run() int {
	if a.onErr == nil {
		a.onErr = func(err error) {
			router.DefaultRouterErrorDisplay(err)
			a.renderBases()
		}
	}
	if a.beforeLoad != nil {
		a.beforeLoad()
	}
	a.Router.OnError(a.onErr)
	a.Router.Run()
	if a.onLoad != nil {
		a.onLoad()
	}
	// Get the preloader, remove it if it exists
	if preloader, err := jsext.QuerySelector("#" + JSEXT_PRELOADER_ID); err == nil {
		preloader.Remove()
	}
	<-WAITER
	return 0
}

// Exit the application.
func (a *Application) Stop() {
	Exit()
}

// Run the application loader for a time consuming function.
func (a *Application) Load(f func()) {
	if a.Loader != nil && f != nil {
		a.Loader.Show()
		go func() {
			f()
			a.Loader.Finalize()
		}()
	}
}

// Register routes to the application.
func (a *Application) Register(name string, hashOrPath string, callable func(a *Application, v vars.Vars, u *url.URL)) *routes.Route {
	var ncall func(v vars.Vars, u *url.URL)
	if callable != nil {
		ncall = a.WrapURL(callable)
	}
	var route = a.Router.Register(name, hashOrPath, ncall)
	return route
}

func (a *Application) WrapURL(f func(a *Application, v vars.Vars, u *url.URL)) func(v vars.Vars, u *url.URL) {
	return func(v vars.Vars, u *url.URL) {
		if f != nil {
			f(a, v, u)
		}
	}
}

// Renders components of the following types to the application:
//
//   - jsext.Value
//   - jsext.Element
//   - components.Component
//   - js.Value
//   - string
func (a *Application) Render(e ...any) {
	a.Base.InnerHTML("")
	a.appendAny(e...)
	a.renderBases()
}

// Append components of the following types to the application:
//
//   - jsext.Value
//   - jsext.Element
//   - components.Component
//   - js.Value
//   - string
func (a *Application) appendAny(e ...any) {
	for _, el := range e {
		switch el := el.(type) {
		case jsext.Value:
			a.Base.AppendChild(jsext.Element(el))
		case jsext.Element:
			a.Base.AppendChild(el)
		case components.Component:
			a.Base.AppendChild(el.Render())
		case js.Value:
			a.Base.AppendChild(jsext.Element(el))
		case string:
			var oldHTML = a.Base.Get("innerHTML")
			a.Base.Set("innerHTML", oldHTML.String()+el)
		}
	}
}

// insertBefore a list of components to the application.
func (a *Application) insertBefore(before jsext.Element, e ...any) {
	for _, el := range e {
		switch el := el.(type) {
		case jsext.Value:
			a.Base.InsertBefore(jsext.Element(el), before)
		case jsext.Element:
			a.Base.InsertBefore(el, before)
		case components.Component:
			a.Base.InsertBefore(el.Render(), before)
		case js.Value:
			a.Base.InsertBefore(jsext.Element(el), before)
		}
	}
}

// Redirect to a url.
func (a *Application) Redirect(url string) {
	a.Router.Redirect(url)
}

// InnerHTML sets the inner HTML of the element.
func (a *Application) RenderHTML(html string) *Application {
	a.Base.InnerHTML("")
	a.Base.InnerHTML(html)
	a.renderBases()
	return a
}

// InnerText sets the inner text of the element.
func (a *Application) RenderText(text string) *Application {
	a.Base.InnerHTML("")
	a.Base.InnerText(text)
	a.renderBases()
	return a
}

// AppendChild appends a child to the element.
// Can render the following types:
//
//   - jsext.Value
//   - jsext.Element
//   - components.Component
//   - js.Value
func (a *Application) AppendChild(e ...any) *Application {
	// If footer is not nil, append before it
	if a.Footer != nil {
		var footer, ok = a.Footer.(*elements.Element)
		if !ok {
			panic("footer is not an element, cannot append before it.")
		}
		a.insertBefore(footer.Render(), e...)
	} else {
		a.appendAny(e...)
	}
	return a
}

// Render application header and footer if defined.
func (a *Application) renderBases() {
	if a.Navbar != nil {
		a.Base.Prepend(a.Navbar.Render())
	}
	if a.Footer != nil {
		a.Base.Append(a.Footer.Render())
	}
}

// Exit the application.
func Exit() {
	WAITER <- struct{}{}
	close(WAITER)
}
