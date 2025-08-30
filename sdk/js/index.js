const fetchFn = typeof fetch !== 'undefined' ? fetch : require('node-fetch');

class Client {
  constructor({ baseURL = 'https://api.supplychain.example.com', apiKey, clientType = 'agent' } = {}){
    this.baseURL = baseURL;
    this.apiKey = apiKey;
    this.clientType = clientType;
  }
  _headers(extra={}){
    return Object.assign({ 'Authorization': `Bearer ${this.apiKey}`, 'X-Client-Type': this.clientType }, extra);
  }
  async alerts(params={}){
    const qs = new URLSearchParams(params).toString();
    const resp = await fetchFn(`${this.baseURL}/v1/alerts?${qs}`, { headers: this._headers() });
    return resp.json();
  }
  async usage(){
    const resp = await fetchFn(`${this.baseURL}/v1/usage`, { headers: this._headers() });
    return resp.json();
  }
  async usageTimeseries({bucket='day', start, end}={}){
    const params = new URLSearchParams({ bucket });
    if (start) params.set('start', start);
    if (end) params.set('end', end);
    const resp = await fetchFn(`${this.baseURL}/v1/usage/timeseries?${params}`, { headers: this._headers() });
    return resp.json();
  }
  async createCheckoutSession({planCode, interval='month', overageEnabled=false}){
    const resp = await fetchFn(`${this.baseURL}/v1/billing/checkout-session`, {
      method: 'POST',
      headers: this._headers({ 'Content-Type': 'application/json' }),
      body: JSON.stringify({ plan_code: planCode, interval, overage_enabled: overageEnabled })
    });
    return (await resp.json()).url;
  }
  async createPortalSession(){
    const resp = await fetchFn(`${this.baseURL}/v1/billing/portal-session`, { method: 'POST', headers: this._headers() });
    return (await resp.json()).url;
  }
}

module.exports = { Client };
