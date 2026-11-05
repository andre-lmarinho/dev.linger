package tray

import (
	"fmt"
	"net"
	"os"
	"strconv"
	"strings"
	"sync"

	"github.com/devlinger/devlinger/internal/monitor"
	"github.com/godbus/dbus/v5"
	"github.com/godbus/dbus/v5/prop"
)

const itemPath = dbus.ObjectPath("/StatusNotifierItem")
const menuPath = dbus.ObjectPath("/MenuBar")

type Tray struct {
	conn     *dbus.Conn
	props    *prop.Properties
	mu       sync.RWMutex
	services []monitor.Service
	revision uint32
	quit     chan struct{}
	once     sync.Once
}

type pixmap struct {
	Width, Height int32
	Data          []byte
}
type layout struct {
	ID         int32
	Properties map[string]dbus.Variant
	Children   []dbus.Variant
}
type menuItem struct {
	ID         int32
	Properties map[string]dbus.Variant
}
type event struct {
	ID        int32
	EventID   string
	Data      dbus.Variant
	Timestamp uint32
}

func New() (*Tray, error) {
	conn, err := dbus.ConnectSessionBus()
	if err != nil {
		return nil, err
	}
	t := &Tray{conn: conn, quit: make(chan struct{})}
	fail := func(err error) (*Tray, error) { conn.Close(); return nil, err }
	name := fmt.Sprintf("org.kde.StatusNotifierItem-%d-1", os.Getpid())
	reply, err := conn.RequestName(name, dbus.NameFlagDoNotQueue)
	if err != nil {
		return fail(err)
	}
	if reply != dbus.RequestNameReplyPrimaryOwner {
		return fail(fmt.Errorf("indicator name already owned"))
	}
	if err = conn.Export(t, itemPath, "org.kde.StatusNotifierItem"); err != nil {
		return fail(err)
	}
	if err = conn.Export(t, menuPath, "com.canonical.dbusmenu"); err != nil {
		return fail(err)
	}
	t.props, err = prop.Export(conn, itemPath, map[string]map[string]*prop.Prop{"org.kde.StatusNotifierItem": {
		"Category": {Value: "ApplicationStatus", Emit: prop.EmitFalse}, "Id": {Value: "devlinger", Emit: prop.EmitFalse},
		"Title": {Value: "DevLinger", Emit: prop.EmitTrue}, "Status": {Value: "Active", Emit: prop.EmitFalse},
		"IconName": {Value: "", Emit: prop.EmitFalse}, "IconPixmap": {Value: trayIcon(false), Emit: prop.EmitFalse},
		"Menu": {Value: menuPath, Emit: prop.EmitFalse}, "ItemIsMenu": {Value: true, Emit: prop.EmitFalse},
	}})
	if err != nil {
		return fail(err)
	}
	_, err = prop.Export(conn, menuPath, map[string]map[string]*prop.Prop{"com.canonical.dbusmenu": {
		"Version": {Value: uint32(3), Emit: prop.EmitFalse}, "TextDirection": {Value: "ltr", Emit: prop.EmitFalse},
		"Status": {Value: "normal", Emit: prop.EmitFalse}, "IconThemePath": {Value: []string{}, Emit: prop.EmitFalse},
	}})
	if err != nil {
		return fail(err)
	}
	call := conn.Object("org.kde.StatusNotifierWatcher", "/StatusNotifierWatcher").Call("org.kde.StatusNotifierWatcher.RegisterStatusNotifierItem", 0, name)
	if call.Err != nil {
		return fail(fmt.Errorf("register system tray indicator: %w", call.Err))
	}
	return t, nil
}

func trayIcon(active bool) []pixmap {
	icon := make([]byte, 16*16*4)
	for y := 0; y < 16; y++ {
		for x := 0; x < 16; x++ {
			if x >= 2 && x <= 13 && y >= 3 && y <= 12 {
				i := (y*16 + x) * 4
				icon[i], icon[i+1], icon[i+2], icon[i+3] = 255, 62, 185, 155
				if !active {
					icon[i], icon[i+1], icon[i+2], icon[i+3] = 160, 140, 140, 140
				}
				if y == 6 || y == 9 {
					icon[i+1], icon[i+2], icon[i+3] = 235, 255, 250
				}
			}
		}
	}
	return []pixmap{{16, 16, icon}}
}

func (t *Tray) Update(services []monitor.Service, scanErr error) {
	t.mu.Lock()
	t.services = append([]monitor.Service(nil), services...)
	if scanErr != nil {
		t.services = []monitor.Service{{Name: "Unable to read listeners"}}
	}
	t.revision++
	rev := t.revision
	t.mu.Unlock()
	t.props.SetMust("org.kde.StatusNotifierItem", "Title", fmt.Sprintf("DevLinger — %d services", len(services)))
	t.props.SetMust("org.kde.StatusNotifierItem", "IconPixmap", trayIcon(scanErr != nil || len(services) > 0))
	t.conn.Emit(itemPath, "org.kde.StatusNotifierItem.NewIcon")
	t.conn.Emit(itemPath, "org.kde.StatusNotifierItem.NewTitle")
	t.conn.Emit(menuPath, "com.canonical.dbusmenu.LayoutUpdated", rev, int32(0))
}

func (t *Tray) Done() <-chan struct{}                      { return t.quit }
func (t *Tray) Close()                                     { t.conn.Close() }
func (t *Tray) Activate(int32, int32) *dbus.Error          { return nil }
func (t *Tray) SecondaryActivate(int32, int32) *dbus.Error { return nil }
func (t *Tray) ContextMenu(int32, int32) *dbus.Error       { return nil }
func (t *Tray) Scroll(int32, string) *dbus.Error           { return nil }

func (t *Tray) items() []layout {
	items := []layout{{ID: 1, Properties: map[string]dbus.Variant{"label": dbus.MakeVariant(fmt.Sprintf("DevLinger — %d services", len(t.services))), "enabled": dbus.MakeVariant(false)}}}
	if len(t.services) == 0 {
		items = append(items, layout{ID: 2, Properties: map[string]dbus.Variant{"label": dbus.MakeVariant("No recognized services found"), "enabled": dbus.MakeVariant(false)}})
	}
	childID := int32(len(t.services) + 10)
	for i, s := range t.services {
		item := layout{ID: int32(i + 10), Properties: map[string]dbus.Variant{"label": dbus.MakeVariant(formatService(s)), "enabled": dbus.MakeVariant(false)}, Children: []dbus.Variant{}}
		if len(s.Members) > 0 {
			label := "containers"
			if len(s.Members) == 1 {
				label = "container"
			}
			item.Properties = map[string]dbus.Variant{"label": dbus.MakeVariant(fmt.Sprintf("%s · Docker · %d %s", s.Name, len(s.Members), label)), "children-display": dbus.MakeVariant("submenu")}
			for _, member := range s.Members {
				child := layout{ID: childID, Properties: map[string]dbus.Variant{"label": dbus.MakeVariant(formatService(monitor.Service{Name: member.Name, Endpoints: member.Endpoints})), "enabled": dbus.MakeVariant(false)}, Children: []dbus.Variant{}}
				item.Children = append(item.Children, dbus.MakeVariant(child))
				childID++
			}
		}
		items = append(items, item)
	}
	items = append(items, layout{ID: 3, Properties: map[string]dbus.Variant{"type": dbus.MakeVariant("separator")}}, layout{ID: 4, Properties: map[string]dbus.Variant{"label": dbus.MakeVariant("Quit")}})
	for i := range items {
		if items[i].Children == nil {
			items[i].Children = []dbus.Variant{}
		}
	}
	return items
}

func formatService(s monitor.Service) string {
	parts := []string{s.Name}
	var members []string
	for _, member := range s.Members {
		members = append(members, formatService(monitor.Service{Name: member.Name, Endpoints: member.Endpoints}))
	}
	if len(members) > 0 {
		parts = append(parts, strings.Join(members, ", "))
	}
	for _, endpoint := range s.Endpoints {
		parts = append(parts, net.JoinHostPort(endpoint.Address, strconv.Itoa(int(endpoint.Port))))
	}
	if s.Process != "" && s.Process != s.Name {
		parts = append(parts, s.Process)
	}
	if s.PID != 0 {
		parts = append(parts, fmt.Sprintf("PID %d", s.PID))
	}
	if s.Project != "" {
		parts = append(parts, s.Project)
	}
	return strings.Join(parts, " · ")
}

func (t *Tray) GetLayout(parent int32, depth int32, properties []string) (uint32, layout, *dbus.Error) {
	t.mu.RLock()
	defer t.mu.RUnlock()
	root := layout{ID: 0, Properties: map[string]dbus.Variant{"children-display": dbus.MakeVariant("submenu")}, Children: []dbus.Variant{}}
	for _, item := range t.items() {
		root.Children = append(root.Children, dbus.MakeVariant(item))
	}
	if parent != 0 {
		found, ok := findItem(root, parent)
		if !ok {
			return 0, root, dbus.MakeFailedError(fmt.Errorf("unknown menu item"))
		}
		root = found
	}
	return t.revision, limitDepth(root, depth), nil
}

func findItem(item layout, id int32) (layout, bool) {
	if item.ID == id {
		return item, true
	}
	for _, child := range item.Children {
		if found, ok := findItem(child.Value().(layout), id); ok {
			return found, true
		}
	}
	return layout{}, false
}

func limitDepth(item layout, depth int32) layout {
	if depth == 0 {
		item.Children = []dbus.Variant{}
		return item
	}
	for i, child := range item.Children {
		next := depth
		if depth > 0 {
			next--
		}
		item.Children[i] = dbus.MakeVariant(limitDepth(child.Value().(layout), next))
	}
	return item
}

func (t *Tray) GetGroupProperties(ids []int32, names []string) ([]menuItem, *dbus.Error) {
	t.mu.RLock()
	defer t.mu.RUnlock()
	var result []menuItem
	root := layout{ID: 0}
	for _, item := range t.items() {
		root.Children = append(root.Children, dbus.MakeVariant(item))
	}
	for _, id := range ids {
		if item, ok := findItem(root, id); ok {
			result = append(result, menuItem{item.ID, item.Properties})
		}
	}
	if result == nil {
		result = []menuItem{}
	}
	return result, nil
}
func (t *Tray) GetProperty(id int32, name string) (dbus.Variant, *dbus.Error) {
	t.mu.RLock()
	defer t.mu.RUnlock()
	root := layout{ID: 0}
	for _, item := range t.items() {
		root.Children = append(root.Children, dbus.MakeVariant(item))
	}
	if item, ok := findItem(root, id); ok {
		if value, ok := item.Properties[name]; ok {
			return value, nil
		}
	}
	return dbus.Variant{}, dbus.MakeFailedError(fmt.Errorf("unknown property"))
}
func (t *Tray) Event(id int32, eventID string, data dbus.Variant, timestamp uint32) *dbus.Error {
	if id == 4 && eventID == "clicked" {
		t.once.Do(func() { close(t.quit) })
	}
	return nil
}
func (t *Tray) EventGroup(events []event) ([]int32, *dbus.Error) {
	for _, e := range events {
		t.Event(e.ID, e.EventID, e.Data, e.Timestamp)
	}
	return []int32{}, nil
}
func (t *Tray) AboutToShow(int32) (bool, *dbus.Error) { return false, nil }
func (t *Tray) AboutToShowGroup(ids []int32) ([]int32, []int32, *dbus.Error) {
	return []int32{}, []int32{}, nil
}
