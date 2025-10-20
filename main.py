import csv
import difflib
import os
import json

from kivy.app import App
from kivy.uix.popup import Popup
from kivy.uix.screenmanager import ScreenManager, Screen
from kivy.uix.boxlayout import BoxLayout
from kivy.uix.textinput import TextInput
from kivy.uix.button import Button
from kivy.uix.label import Label
from kivy.uix.scrollview import ScrollView
from kivy.uix.gridlayout import GridLayout
from kivy.clock import Clock
from kivy.graphics import Color, Rectangle
from kivy.animation import Animation

from src.commitmgr import JsonCommitQueue
from src.server import SupabaseAPI

os.environ["KIVY_NO_CONSOLELOG"] = "1"
os.environ["KIVY_LOG_MODE"] = "error"

from kivy.config import Config
Config.set('kivy', 'log_level', 'error')

global api
global commit_queue

needs_key = False
if os.path.isfile("settings.json"):
    with open("settings.json", "r") as f:
        try:
            file = json.load(f)
            SUPABASE_KEY = file["supabase_key"]
            SUPABASE_URL = file["supabase_url"]
            api = SupabaseAPI(SUPABASE_URL, SUPABASE_KEY)
            commit_queue = JsonCommitQueue(api)

            print(api.check())
            f.close()
        except:
            f.close()
            with open("settings.json", "w") as file:
                base = {"supabase_url": "", "supabase_key": ""}
                file.close()
                needs_key = True
else:
    with open("settings.json", "w") as file:
        base = {"supabase_url": "", "supabase_key": ""}
        file.close()
        needs_key = True

# api = SupabaseAPI(SUPABASE_URL, SUPABASE_KEY)
# commit_queue = JsonCommitQueue(api)


class ErrorBar(BoxLayout):
    def __init__(self, message, **kwargs):
        super().__init__(orientation='vertical', size_hint_y=None, height=40, **kwargs)

        self.label = Label(text=message, color=(1, 1, 1, 1))
        self.add_widget(self.label)

        # Draw background
        with self.canvas.before:
            Color(1, 0, 0, 1)  # Red background
            self.rect = Rectangle(size=self.size, pos=self.pos)

        # Update bg size on resize
        self.bind(size=self._update_rect, pos=self._update_rect)

        # Auto fade-out
        Clock.schedule_once(self.fade_and_remove, 3)

    def _update_rect(self, *args):
        self.rect.pos = self.pos
        self.rect.size = self.size

    def fade_and_remove(self, *args):
        anim = Animation(opacity=0, duration=0.5)
        anim.bind(on_complete=lambda *x: self.parent.remove_widget(self) if self.parent else None)
        anim.start(self)



class CommitUI(BoxLayout):
    """
    freaky freak code
    """
    def __init__(self, **kwargs):
        """
        Initialise the app.
        """
        super().__init__(orientation='vertical', **kwargs)

        self.scanner_input = TextInput(hint_text="Scan location code...", multiline=False, size_hint_y=None, height=50)
        self.scanner_input.bind(on_text_validate=self.on_scanned)
        self.add_widget(self.scanner_input)

        self.location_label = Label(text="Location: (waiting for scan)")
        self.add_widget(self.location_label)

        self.delta_input = TextInput(hint_text="Enter quantity", multiline=False, size_hint_y=None, height=50, input_filter='int')
        self.add_widget(self.delta_input)

        btn_layout = BoxLayout(size_hint_y=None, height=60)
        self.toggle_btn = Button(text="Mode: ADD")
        self.toggle_btn.bind(on_release=self.toggle_mode)
        btn_layout.add_widget(self.toggle_btn)

        self.commit_btn = Button(text="Commit")
        self.commit_btn.bind(on_release=self.commit)
        btn_layout.add_widget(self.commit_btn)

        self.change_item_btn = Button(text="Change Item")
        self.change_item_btn.bind(on_release=self.show_item_search)
        btn_layout.add_widget(self.change_item_btn)

        self.add_widget(btn_layout)
        self.mode = "ADD"
        self.location = None
        self.item_id = None
        self.locations = self.load_locations()
        self.items = self.load_items()
        self.scanner_input.focus = True

    def load_items(self):
        """
        Try to fetch the items and their ids from the server, and return it.
        if it fails, get the last cached data from the csv.
        """
        items = {}
        try:
            api.export_items_to_csv("items.csv")
        except:
            print("Could not fetch new items data. Using cached data.")

        with open("items.csv", newline='', encoding="utf-8") as f:
            reader = csv.DictReader(f)
            for row in reader:
                items[row['name']] = int(row['id'])
        return items


    def load_locations(self):
        """
        Try to fetch the locations and the item id's in them from the server, and return it.
        if it fails, get the last cached data from the csv.
        """
        locations = {}
        try:
            api.export_location_data_to_csv("locations.csv")
        except:
            print("Could not fetch new location data. Using cached data.")

        with open("locations.csv", newline='', encoding="utf-8") as f:
            reader = csv.DictReader(f)
            for row in reader:
                locations[row['location']] = row['items']
        return locations


    def on_scanned(self, instance):
        """
        Logic for when we scan a location tag.
        Tries to look for the items associated with the location.
        If there is more than one item, open a popup to ask which one to use
        If there is no items, open a popup with a searchbox to select an item.
        """
        self.locations = self.load_locations()

        self.location = instance.text.strip()
        instance.text = ""

        if self.location in self.locations.keys():
            items = self.locations[self.location].strip("[]").split(", ")
            items = [int(i) for i in items if i.strip()]

            if len(items) > 1:
                print("More than one item for given location!")

                # Popup box to allow the user to choose which one.
                popup_layout = BoxLayout(orientation='vertical', spacing=10, padding=10)
                popup_layout.add_widget(Label(text=f"Select item for {self.location}:"))
                for i in items:
                    name = next((k for k, v in self.items.items() if int(v) == i), None)

                    if name == None:
                        text = f"No item found for ID: {i}"
                    else:
                        text = name

                    btn = Button(text=text, size_hint_y=None, height=50)
                    def select_item(instance, item=i):
                        self.item_id = int(item)
                        self.location_label.text = f"Location: {self.location}\nItem: {self.item_id}"
                        popup.dismiss()
                    btn.bind(on_release=select_item)
                    popup_layout.add_widget(btn)

                popup = Popup(title="Multiple items found",
                            content=popup_layout,
                            size_hint=(0.8, 0.6),
                            auto_dismiss=True)
                popup.open()
                return

            else:
                item = items[0]

            self.item_id = item
            item_id_text = f"Item: {self.item_id}"
        else:
            self.build_item_search_ui(
                title=f"No item found for {self.location}",
                on_select_callback=lambda name: self._set_item_by_name(name)
            )
            item_id_text = f"Item: {self.item_id}"

        self.location_label.text = f"Location: {self.location}\n{item_id_text}"


    def toggle_mode(self, instance):
        """
        Change the mode from add to subtract
        """
        self.mode = "SUB" if self.mode == "ADD" else "ADD"
        self.toggle_btn.text = f"Mode: {self.mode}"


    def commit(self, instance):
        """
        Logic to commit a change to the server.
        """
        if not self.location or not self.item_id:
            print("No location or item selected")
            self.scanner_input.focus = True
            return
        try:
            qty = int(self.delta_input.text)
            if self.mode == "SUB":
                qty = -qty
            commit_queue.submit_commit("TOUGHPAD01", self.location, qty, self.item_id)
            self.delta_input.text = ""
            print("Commit queued")
        except ValueError:
            print("Invalid number")
        self.scanner_input.focus = True


    def show_item_search(self, instance):
        """
        Open the item select ui
        """
        self.build_item_search_ui(
            title="Select an Item",
            on_select_callback=lambda name: self._set_item_by_name(name)
        )


    def _set_item_by_name(self, name):
        """
        Helper function for build_item_search_ui
        """
        self.item_id = self.items[name]
        self.location_label.text = f"Location: {self.location}\nItem: {self.item_id}"


    def build_item_search_ui(self, title: str, on_select_callback):
        """
        Create the item selection UI for searching for and selecting items.
        """
        layout = BoxLayout(orientation='vertical', spacing=10, padding=10)

        search_input = TextInput(hint_text="Search items...", multiline=False, size_hint_y=None, height=50)
        results_layout = GridLayout(cols=1, size_hint_y=None)
        results_layout.bind(minimum_height=results_layout.setter('height'))
        scroll = ScrollView(size_hint=(1, 1))
        scroll.add_widget(results_layout)

        layout.add_widget(search_input)
        layout.add_widget(scroll)

        popup = Popup(title=title, content=layout, size_hint=(0.9, 0.9), auto_dismiss=True)

        def update_results(text):
            results_layout.clear_widgets()
            matches = difflib.get_close_matches(text, self.items.keys(), n=15, cutoff=0.1)
            for name in matches:
                btn = Button(text=name, size_hint_y=None, height=40)
                btn.bind(on_release=lambda _, n=name: (on_select_callback(n), popup.dismiss()))
                results_layout.add_widget(btn)

        search_input.bind(text=lambda _, t: update_results(t))
        popup.open()


    def build_main_ui(self):
        self.clear_widgets()
        self.__init__()


class WelcomeScreen(Screen):
    def __init__(self, **kwargs):
        super().__init__(**kwargs)

        def onrelease(state):
            def _switch(instance):
                self.manager.current = state
            return _switch

        layout = BoxLayout(orientation='vertical', spacing=20, padding=50)
        layout.add_widget(Button(text="Add/Remove Stock", on_release=onrelease('commit')))
        # layout.add_widget(Button(text="Stock Take", on_release=onrelease('stock')))
        layout.add_widget(Button(text="Exit", on_release=lambda x: App.get_running_app().stop()))
        self.add_widget(layout)


class Settings():
    pass


class CommitScreen(Screen):
    def __init__(self, **kwargs):
        super().__init__(**kwargs)
        layout = BoxLayout(orientation='vertical')

        back_btn = Button(text="Back to Main Menu", size_hint_y=None, height=60)
        back_btn.bind(on_release=lambda x: setattr(self.manager, 'current', 'welcome'))
        layout.add_widget(back_btn)

        self.add_widget(layout)

        layout.add_widget(CommitUI())


class KeyInputUI(BoxLayout):
    def __init__(self, sm, **kwargs):
        super().__init__(orientation='vertical', **kwargs)

        self.sm = sm

        # layout = BoxLayout( spacing=20, padding=50)
        self.url_input = TextInput(hint_text="Input API URL", multiline=False, size_hint_y=None, height=50)
        self.url_input.bind(on_text_validate=self.on_input_url)
        self.add_widget(self.url_input)

        self.key_input = TextInput(hint_text="Input API Key", multiline=False, size_hint_y=None, height=50)
        self.key_input.bind(on_text_validate=self.on_input_key)
        self.add_widget(self.key_input)

        self.submit_button = Button(text = "Submit")
        self.submit_button.bind(on_release=self.submit)
        self.add_widget(self.submit_button)

    def on_input_url(self, instance):
        self.key_input.focus = True

    def on_input_key(self, instance):
        self.submit(self)

    def show_error(self, message):
        # Remove existing error bars
        for child in list(self.children):
            if isinstance(child, ErrorBar):
                self.remove_widget(child)

        # Add new one at top
        self.add_widget(ErrorBar(message), index=0)

    def submit(self, instance):
        self.url = self.url_input.text.strip()
        self.key = self.key_input.text.strip()

        print("Connecting to SupaBase...")
        print(f"key: {self.key}")
        print(f"url: {self.url}")

        global api
        global commit_queue

        if not self.url.startswith("http"):
            self.url = f"https://{self.url}.supabase.co"

        api = SupabaseAPI(self.url, self.key)
        commit_queue = JsonCommitQueue(api)

        valid = api.check()

        if valid:
            obj = {"supabase_url": self.url, "supabase_key": self.key}
            with open("settings.json", "w") as f:
                json.dump(obj, f)
            self.sm.current = 'welcome'
        else:
            self.show_error("Invalid credentials or cannot connect.")
            return


class KeyInputScreen(Screen):
    def __init__(self, sm, **kwargs):
        super().__init__(**kwargs)
        layout = BoxLayout(orientation='vertical')
        self.add_widget(layout)
        layout.add_widget(KeyInputUI(sm=sm))



class CommitApp(App):
    def build(self):
        sm = ScreenManager()
        sm.add_widget(WelcomeScreen(name='welcome'))
        sm.add_widget(CommitScreen(name='commit'))
        sm.add_widget(KeyInputScreen(sm, name='keyin'))
        # Add more screens as needed:
        # sm.add_widget(OtherScreen(name='other'))

        sm.current = 'keyin' if needs_key else 'welcome'
        return sm


if __name__ == "__main__":
    CommitApp().run()
