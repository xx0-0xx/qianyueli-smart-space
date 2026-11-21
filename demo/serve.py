"""Read-only UI fixture server. Not the Go backend and never controls devices."""
from http.server import SimpleHTTPRequestHandler, ThreadingHTTPServer
from pathlib import Path
from urllib.parse import parse_qs, urlsplit
import json
import threading
import time

ROOT = Path(__file__).resolve().parents[1]
NOW = int(time.time() * 1000)
STORES = [
    dict(id=101, name="阡月里 · 湖畔演示店", address="演示地址 · 不对应真实门店", phone="", status=1,
         invite_code="DEMO101", balance_group_id=101, auto_approve_refund=True, description="虚构门店，仅供界面展示"),
    dict(id=102, name="阡月里 · 云栖演示店", address="演示地址 · 不对应真实门店", phone="", status=1,
         invite_code="DEMO102", balance_group_id=102, auto_approve_refund=True, description="虚构门店，仅供界面展示"),
]
TYPES = [dict(id=201, store_id=101, name="雅致包间", description="4 人棋牌包间", sort=1),
         dict(id=202, store_id=101, name="观景包间", description="宽敞独立包间", sort=2)]
DEVICES = [dict(id=401+i, store_id=101, resource_id=301+i//2, name=("灯光" if i%2==0 else "门锁")+f" · {i//2+1}号包间",
                device_sn=f"DEMO-DEVICE-{i+1:02}", type="light" if i%2==0 else "door_lock", status=1,
                config_json=json.dumps(dict(cmd_on="a1",cmd_off="b1",is_toggle=True))) for i in range(12)]
RESOURCES = [dict(id=301+i, store_id=101, resource_type_id=201+i%2, name=f"{['听雨','观澜','云栖','竹影','清风','明月'][i]}包间",
                  status=2 if i<2 else 0, sort=i, image="", device_ids=json.dumps([401+2*i,402+2*i]),
                  resource_type=TYPES[i%2]) for i in range(6)]
ORDERS = [dict(id=501+i, order_no=f"DEMO202611{501+i}", store_id=101, user_id=601+i%2, resource_id=301+i,
              resource_name=RESOURCES[i]['name'], user_nickname=f"演示顾客{1+i%2}", nickname=f"演示顾客{1+i%2}",
              status=[2,2,3,0][i], start_time=NOW-3600000*(i+1), end_time=NOW+3600000 if i<2 else NOW-1800000,
              original_amount=10000, member_discount=1500, total_amount=8500, paid_amount=8500, pay_method="balance",
              created_at=NOW-14400000, store=STORES[0], resource=RESOURCES[i]) for i in range(4)]
ORDERS[3].update(start_time=NOW+7200000, end_time=NOW+10800000)
for order in ORDERS:
    order['duration']=(order['end_time']-order['start_time'])//60000
LEVELS = [dict(id=701, store_id=101, name="银卡会员", discount_bp=9500, status=1, sort=1),
          dict(id=702, store_id=101, name="金卡会员", discount_bp=8500, status=1, sort=2)]
CUSTOMERS = [dict(id=601+i, nickname=f"演示顾客{i+1}", avatar="", phone="未提供（演示）", order_count=2,
                  total_spent=17000, last_order_at=NOW, level_id=701+i, level_name=LEVELS[i]['name'],
                  level_status=1, member_status=1, expires_at=None, remark="虚构数据") for i in range(2)]

def page(rows, query):
    rows = list(rows)
    keyword = (query.get('keyword') or [''])[0].strip()
    if keyword:
        rows = [row for row in rows if keyword in json.dumps(row, ensure_ascii=False)]
    if (query.get('status') or [''])[0] != '':
        rows = [row for row in rows if str(row.get('status')) == query['status'][0]]
    try:
        number = max(1, int(query.get('page', ['1'])[0]))
        size = min(500, max(1, int((query.get('page_size') or query.get('size') or ['20'])[0])))
    except ValueError:
        number, size = 1, 20
    return dict(list=rows[(number-1)*size:number*size], total=len(rows), page=number,
                page_size=size, total_pages=max(1, (len(rows)+size-1)//size))

class API(SimpleHTTPRequestHandler):
    def log_message(self, fmt, *args):
        print('[fixture-api] ' + fmt % args, flush=True)

    def reply(self, data=None, status=200, message='ok'):
        raw = json.dumps(dict(code=0 if status==200 else status, message=message, data=data), ensure_ascii=False).encode()
        self.send_response(status)
        origin = self.headers.get('Origin', '')
        if origin in {'http://localhost:5173','http://localhost:5174','http://127.0.0.1:5173','http://127.0.0.1:5174'}:
            self.send_header('Access-Control-Allow-Origin', origin)
            self.send_header('Vary', 'Origin')
        self.send_header('Access-Control-Allow-Headers','Content-Type, Authorization, X-Store-Id')
        self.send_header('Access-Control-Allow-Methods','GET, POST, OPTIONS')
        self.send_header('Content-Type','application/json; charset=utf-8')
        self.send_header('Content-Length',str(len(raw)))
        self.send_header('Cache-Control','no-store')
        self.end_headers()
        self.wfile.write(raw)

    def do_OPTIONS(self):
        self.reply()

    def do_POST(self):
        path = urlsplit(self.path).path
        try:
            length=int(self.headers.get('Content-Length','0'))
            if length>4096: return self.reply(status=413,message='Request too large')
            data=json.loads(self.rfile.read(length) or b'{}')
        except (ValueError, UnicodeError):
            return self.reply(status=400,message='Invalid JSON')
        admin=path in {'/api/v1/admin/login','/api/v1/admin/login/web'}
        merchant=path in {'/api/v1/merchant/invite-login','/api/v1/merchant/invite-login/web'}
        if not (admin or merchant):
            return self.reply(status=405,message='只读界面演示：不执行保存、退款或设备指令')
        valid=(data.get('username')=='demo' and data.get('password')=='demo') if admin else data.get('invite_code')=='DEMO101'
        if not valid:
            return self.reply(status=401,message='演示登录：平台 demo / demo，商家邀请码 DEMO101')
        self.reply(dict(access_token='demo-only-not-a-jwt',user=dict(id=1,nickname='界面演示',role='super_admin' if admin else 'merchant',store_id=101),store_id=101,store=STORES[0]))

    def do_PUT(self):
        self.reply(status=405,message='只读界面演示：不执行修改')

    do_DELETE=do_PUT
    do_PATCH=do_PUT

    def do_GET(self):
        parts=urlsplit(self.path)
        path=parts.path.removeprefix('/api/v1')
        query=parse_qs(parts.query)
        routes={
            '/admin/statistics/overview':dict(total_stores=2,total_users=2,total_orders=4,total_revenue=255),
            '/admin/stores':page(STORES,query), '/merchant/stores':STORES,
            '/admin/balance-groups':[dict(id=s['id'],name=s['name']) for s in STORES],
            '/admin/devices':page(DEVICES,query), '/merchant/devices':DEVICES,
            '/merchant/resources':RESOURCES, '/stores/101/resources':RESOURCES,
            '/merchant/resource-types':TYPES, '/merchant/orders':page(ORDERS,query),
            '/merchant/customers':page(CUSTOMERS,query), '/merchant/member-levels':LEVELS,
            '/merchant/recharges':[], '/merchant/refund-requests':[], '/merchant/coupons':[],
            '/admin/audio':[], '/merchant/audio-configs':[], '/merchant/config/price-rules':[],
            '/admin/product-templates':[], '/merchant/product-templates':[],
            '/admin/leads':page([],query), '/admin/elys/bindings':[], '/admin/elys/verifications':page([],query),
            '/admin/voice-config':dict(devices=[]), '/merchant/voice-config':dict(devices=[]),
            '/merchant/voice-devices':[], '/merchant/store-config':dict(store=STORES[0]),
        }
        for c in CUSTOMERS:
            routes[f"/merchant/customers/{c['id']}"] = dict(c, orders=ORDERS, recharges=[])
        if path not in routes:
            return self.reply(status=404,message='该接口未纳入界面演示')
        self.reply(routes[path])

def frontend(directory):
    root=(ROOT/directory/'dist').resolve()
    class Static(SimpleHTTPRequestHandler):
        def __init__(self,*args,**kwargs):
            super().__init__(*args,directory=str(root),**kwargs)

        def do_GET(self):
            path=urlsplit(self.path).path
            target=(root/path.lstrip('/')).resolve()
            if not target.is_relative_to(root):
                return self.send_error(403)
            if path=='/' or path.endswith('.html') or (not target.is_file() and '.' not in Path(path).name):
                html=(root/'index.html').read_text(encoding='utf-8')
                banner='<div style="position:fixed;bottom:14px;right:20px;z-index:2147483647;padding:8px 18px;border:1px solid #e6a23c;border-radius:8px;background:#fff7e6;color:#92400e;font:600 13px/1.6 system-ui;pointer-events:none">界面演示 · 虚构数据 · 只读 · 非生产后台</div>'
                raw=html.replace('</body>',banner+'</body>').encode()
                self.send_response(200)
                self.send_header('Content-Type','text/html; charset=utf-8')
                self.send_header('Content-Length',str(len(raw)))
                self.end_headers();self.wfile.write(raw)
            else:
                super().do_GET()
    return Static

def main():
    for module in ['admin-web-v3','admin-merchant-v3']:
        if not (ROOT/module/'dist/index.html').is_file():
            raise SystemExit(f'Build {module} first: pnpm build')
    servers=[]
    try:
        for port,handler in [(8080,API),(5173,frontend('admin-web-v3')),(5174,frontend('admin-merchant-v3'))]:
            servers.append(ThreadingHTTPServer(('127.0.0.1',port),handler))
        for server in servers:
            threading.Thread(target=server.serve_forever,daemon=True).start()
        print('READ-ONLY UI FIXTURES; NOT the Go backend. No devices, payments or real customers.',flush=True)
        print('Platform http://localhost:5173 (demo / demo); Merchant http://localhost:5174 (DEMO101)',flush=True)
        while True: time.sleep(1)
    except KeyboardInterrupt:
        pass
    finally:
        for server in servers: server.server_close()

if __name__=='__main__': main()
