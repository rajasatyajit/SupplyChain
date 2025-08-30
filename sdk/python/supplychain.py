import os, requests

class Client:
    def __init__(self, base_url=None, api_key=None, client_type=None):
        self.base_url = base_url or "https://api.supplychain.example.com"
        self.api_key = api_key or os.environ.get("API_KEY")
        self.client_type = client_type or os.environ.get("CLIENT_TYPE", "agent")
        self.sess = requests.Session()

    def _headers(self):
        h = {"Authorization": f"Bearer {self.api_key}"}
        if self.client_type:
            h["X-Client-Type"] = self.client_type
        return h

    def alerts(self, **params):
        r = self.sess.get(f"{self.base_url}/v1/alerts", headers=self._headers(), params=params)
        r.raise_for_status()
        return r.json()

    def usage(self):
        r = self.sess.get(f"{self.base_url}/v1/usage", headers=self._headers())
        r.raise_for_status()
        return r.json()

    def usage_timeseries(self, bucket="day", start=None, end=None):
        params = {"bucket": bucket}
        if start: params["start"] = start
        if end: params["end"] = end
        r = self.sess.get(f"{self.base_url}/v1/usage/timeseries", headers=self._headers(), params=params)
        r.raise_for_status()
        return r.json()

    def create_checkout_session(self, plan_code, interval="month", overage_enabled=False):
        payload = {"plan_code": plan_code, "interval": interval, "overage_enabled": overage_enabled}
        r = self.sess.post(f"{self.base_url}/v1/billing/checkout-session", headers={**self._headers(), "Content-Type": "application/json"}, json=payload)
        r.raise_for_status()
        return r.json().get("url")

    def create_portal_session(self):
        r = self.sess.post(f"{self.base_url}/v1/billing/portal-session", headers=self._headers())
        r.raise_for_status()
        return r.json().get("url")
