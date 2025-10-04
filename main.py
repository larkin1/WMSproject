import csv
import difflib
from kivy.app import App
from kivy.uix.popup import Popup
from kivy.uix.boxlayout import BoxLayout
from kivy.uix.textinput import TextInput
from kivy.uix.button import Button
from kivy.uix.label import Label
from kivy.uix.scrollview import ScrollView
from kivy.uix.gridlayout import GridLayout
from kivy.clock import Clock

from src.queue import JsonCommitQueue
from src.server import SupabaseAPI

SUPABASE_URL = "https://NOTHING_TO_SEE_HERE.supabase.co"
SUPABASE_KEY = "NOTHING_TO_SEE_HERE"

api = SupabaseAPI(SUPABASE_URL, SUPABASE_KEY)
commit_queue = JsonCommitQueue(api)

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


class CommitApp(App):
    def build(self):
        return CommitUI()


if __name__ == "__main__":
    CommitApp().run()
