import os
import csv

from supabase import create_client, Client

"""
Handle interactions with the server.
Can:
 - Add commits
 - look up commits based on ID
 - export compiled commits to a csv (export an overview)
"""

class SupabaseAPI:
    def __init__(self, url: str, anon_key: str):
        self.supabase: Client = create_client(url, anon_key)

    def send_commit(self, device_id: str, location: str, delta: int, item_id: int):
        payload = {
            "device_id": str(device_id),
            "location": str(location),
            "delta": int(delta),
            "item_id": int(item_id)
        }
        try:
            result = (
                self.supabase.table("commits")
                .insert(payload, returning="representation")
                .execute()
            )
        except Exception as error:
            return {"error": result.error}
        return result.data[0]

    def fetch_commit(self, commit_id):
        """
        Get the data of a particular commit.
        """
        data = (
            self.supabase.table("commits")
                .select("*")
                .eq("commit_id", commit_id)
                .execute()
            )

        return data.data


    def export_overview_to_csv(self, file_path: str = "overview.csv") -> str:
        """
        Exports the live overview (location, item_id, current_qty) into a CSV file.
        Returns the path to the saved file.
        """
        res = self.supabase.from_("overview").select("*").execute()
        rows = res.data or []

        if not rows:
            raise ValueError("No data returned from overview view.")

        with open(file_path, mode="w", newline="", encoding="utf-8") as f:
            writer = csv.DictWriter(f, fieldnames=rows[0].keys())
            writer.writeheader()
            writer.writerows(rows)

        return os.path.abspath(file_path)

    def export_location_data_to_csv(self, file_path: str) -> str:
        """
        Exports the items in each locations to a csv.
        """
        res = self.supabase.from_("locations").select("*").execute()
        rows = res.data or []

        if not rows:
            raise ValueError("No data returned from locations view.")

        with open(file_path, mode="w", newline="", encoding="utf-8") as f:
            writer = csv.DictWriter(f, fieldnames=rows[0].keys())
            writer.writeheader()
            writer.writerows(rows)

        return os.path.abspath(file_path)

    def export_items_to_csv(self, file_path: str) -> str:
        """
        Exports item ids and descriptions to a csv.
        """
        res = self.supabase.from_("items").select("*").execute()
        rows = res.data or []

        if not rows:
            raise ValueError("No data returned from items view.")

        with open(file_path, mode="w", newline="", encoding="utf-8") as f:
            writer = csv.DictWriter(f, fieldnames=rows[0].keys())
            writer.writeheader()
            writer.writerows(rows)

        return os.path.abspath(file_path)



if __name__ == "__main__":
    SUPABASE_URL = "https://NOTHING_TO_SEE_HERE.supabase.co"
    SUPABASE_KEY = "NOTHING_TO_SEE_HERE"

    api = SupabaseAPI(SUPABASE_URL, SUPABASE_KEY)

    resp = api.send_commit(
        device_id="test",
        location="test",
        delta=10,
        item_id=1233
    )
    print(resp)
    commitID = resp["commit_id"]

    out = api.fetch_commit(commitID)
    print(out)

    try:
        csv_path = api.export_overview_to_csv("overview.csv")
        print(f"CSV exported to: {csv_path}")
    except Exception as e:
        print(f"Export failed: {e}")