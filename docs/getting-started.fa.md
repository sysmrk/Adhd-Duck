# آموزش نصب و استفاده ADHD Duck برای شروع کار

این راهنما برای کسی نوشته شده که با kubectl آشناست ولی قبلاً این پروژه را
نصب نکرده است. نسخه کامل انگلیسی با مثال‌های تنظیمات پیشرفته در
[getting-started.md](getting-started.md) قرار دارد.

## ۱. قبل از نصب بدانیم چه چیزی نصب می‌شود

یک controller و روی نودهای انتخاب‌شده یک worker نصب می‌شود. بعد از push شدن
image، CI نام آن را به webhook می‌فرستد. workerها لایه‌ها را زودتر دریافت و
بین نودها توزیع می‌کنند. این قابلیت‌ها از Angry Duck به ارث رسیده‌اند؛
[مقایسه مستند](upstream-comparison.md) تغییرات واقعی این فورک را نشان می‌دهد.

worker به socket مربوط به containerd دسترسی دارد؛ بنابراین از نظر قدرت
کنترل نود باید آن را در سطح root در نظر بگیرید. محدوده فعلی Linux/amd64،
containerd و overlayfs است. این نسخه برای CRI-O/OpenShift یا نود ویندوزی نیست.
برای Production از runtime دارای پشتیبانی استفاده کنید.

روی سیستم مدیریتی Git، kubectl، Helm 3 و OpenSSL لازم است. برای build کردن
imageها Docker و Buildx و یک registry با امکان push نیاز دارید. Kubernetes و
node-exporter باید از قبل نصب باشند. این پروژه آن‌ها را نصب نمی‌کند.

## ۲. کلاستر درست را بررسی کنید

```bash
kubectl config current-context
kubectl get nodes -o wide
kubectl get nodes -o custom-columns=NAME:.metadata.name,RUNTIME:.status.nodeInfo.containerRuntimeVersion,ARCH:.status.nodeInfo.architecture
```

اگر context اشتباه است ادامه ندهید. runtime باید containerd باشد. روی هر
نود انتخابی، وجود socket و مسیر داده را بررسی کنید:

```bash
sudo test -S /run/containerd/containerd.sock
sudo test -d /var/lib/containerd/io.containerd.content.v1.content
sudo containerd config dump | grep -E 'snapshotter|discard_unpacked_layers|config_path'
```

مسیرهای متفاوت در توزیع‌هایی مثل k3s نیاز به تطبیق chart دارند؛ نصب مستقیم
روی آن‌ها را فرض نکنید. برای نگهداری blobها مقدار
`discard_unpacked_layers` باید `false` باشد. فایل پیکربندی containerd را بدون
هماهنگی تیم پلتفرم جایگزین نکنید.

## ۳. سورس و imageها را آماده کنید

```bash
git clone https://github.com/sysmrk/Adhd-Duck.git
cd Adhd-Duck
git rev-parse HEAD
export REGISTRY=registry.example.com/platform
export VERSION=1.9.0
docker login registry.example.com
docker buildx build --platform linux/amd64 --provenance=mode=max --sbom=true -f app/Dockerfile.controller -t "$REGISTRY/adhd-duck-controller:$VERSION" --push app
docker buildx build --platform linux/amd64 --provenance=mode=max --sbom=true -f app/Dockerfile.worker -t "$REGISTRY/adhd-duck-worker:$VERSION" --push app
```

حتماً آدرس نمونه registry را با آدرس واقعی خودتان عوض کنید. digest هر دو image
را از registry بردارید؛ در Production نصب باید به digest ثابت متصل شود.
به صرف وجود تگ یا متن README، منتشرشدن image آماده را فرض نکنید.

## ۴. namespace و توکن‌ها را بسازید

```bash
kubectl create namespace adhdduck --dry-run=client -o yaml | kubectl apply -f -
mkdir -p .local/tokens
chmod 700 .local .local/tokens
umask 077
openssl rand -hex 32 > .local/tokens/shared
openssl rand -hex 32 > .local/tokens/webhook
kubectl -n adhdduck create secret generic adhdduck-rescue-token --from-file=token=.local/tokens/shared --dry-run=client -o yaml | kubectl apply -f -
kubectl -n adhdduck create secret generic adhdduck-webhook-token --from-file=token=.local/tokens/webhook --dry-run=client -o yaml | kubectl apply -f -
```

توکن shared برای ارتباط داخلی و توکن webhook فقط برای CI است. آن‌ها را در
گیت قرار ندهید. در ارتقا این دستورات تولید توکن را مجدداً اجرا نکنید مگر اینکه
عمداً قصد rotation داشته باشید. برای registry خصوصی یک حساب فقط‌خواندنی و
Secret با نام `registry-pull` مطابق بخش ۴ راهنمای انگلیسی بسازید.

## ۵. TLS را آماده کنید

```bash
./scripts/create-peer-tls.sh
kubectl -n adhdduck create secret generic adhdduck-peer-tls --from-file=ca.crt=.local/peer-tls/ca.crt --from-file=tls.crt=.local/peer-tls/tls.crt --from-file=tls.key=.local/peer-tls/tls.key --dry-run=client -o yaml | kubectl apply -f -
```

این اسکریپت یک CA اختصاصی و گواهی ۹۰روزه می‌سازد. کلید CA یعنی `ca.key` باید
آفلاین و محفوظ بماند؛ آن را داخل Kubernetes یا گیت نگذارید. برای محیط سازمانی
گواهی با SAN برابر `adhdduck.internal` و کاربردهای serverAuth/clientAuth را از
PKI اختصاصی سرویس تهیه کنید. قبل از انقضای گواهی آن را تمدید کنید.

## ۶. نصب را از چند نود شروع کنید

نام نودهای نمونه زیر را با نام واقعی عوض کنید:

```bash
kubectl label node worker-01 adhdduck-enabled=true
kubectl label node worker-02 adhdduck-enabled=true
```

فایل `.local/values.yaml` را با محتوای زیر بسازید و registry و digest را اصلاح کنید:

```yaml
controller:
  image:
    repository: registry.example.com/platform/adhd-duck-controller
    tag: "1.9.0"
    digest: "" # digest واقعی controller را بنویسید.
worker:
  image:
    repository: registry.example.com/platform/adhd-duck-worker
    tag: "1.9.0"
    digest: "" # digest واقعی worker را بنویسید.
  nodeSelector:
    adhdduck-enabled: "true"
  tolerations: []
imagePullSecrets:
  - name: registry-pull
ingress:
  enabled: false
serviceMonitor:
  enabled: false
config:
  GC_ENABLED: "false"
  MIRROR_ENABLED: "false"
  ALLOW_SNAPSHOT_TRANSFER: "false"
```

```bash
helm lint ./helm/adhdduck -f .local/values.yaml
helm template adhdduck ./helm/adhdduck -n adhdduck -f .local/values.yaml > .local/rendered.yaml
kubectl apply --dry-run=server -f .local/rendered.yaml
helm upgrade --install adhdduck ./helm/adhdduck -n adhdduck -f .local/values.yaml --wait --timeout 5m
kubectl -n adhdduck get pods -o wide
kubectl -n adhdduck rollout status deployment/adhdduck-controller
kubectl -n adhdduck rollout status daemonset/adhdduck-worker
```

ابتدا dry-run باید موفق شود. اگر admission به hostPath یا root اعتراض کرد،
استثنای محدود مخصوص این سرویس بگیرید؛ کنترل‌های امنیتی کل کلاستر را خاموش نکنید.

## ۷. اولین درخواست را بفرستید

در ترمینال اول:

```bash
kubectl -n adhdduck port-forward service/adhdduck-controller 8080:8080
```

در ترمینال دوم، از پوشه پروژه:

```bash
curl --noproxy '*' --resolve adhdduck.internal:8080:127.0.0.1 --cacert .local/peer-tls/ca.crt https://adhdduck.internal:8080/healthz
curl --fail-with-body --noproxy '*' --resolve adhdduck.internal:8080:127.0.0.1 --cacert .local/peer-tls/ca.crt -H "Authorization: Bearer $(cat .local/tokens/webhook)" -H 'Content-Type: application/json' -d '{"image":"registry.example.com/apps/demo:build-001"}' https://adhdduck.internal:8080/webhook/preheat
```

پاسخ اول باید `ok` باشد. در درخواست دوم نام image واقعی‌ای را بدهید که قبلاً
push کرده‌اید؛ پاسخ مورد انتظار 202 و `accepted:true` است. برای بررسی روند:

```bash
kubectl -n adhdduck logs deployment/adhdduck-controller --tail=100
kubectl -n adhdduck logs -l app.kubernetes.io/component=worker --tail=100 --prefix
```

برای `/status` علاوه بر توکن shared باید `--cert` و `--key` مربوط به TLS را
بدهید؛ مثال دقیق در بخش ۷ راهنمای انگلیسی است. درخواست بدون گواهی باید 401 بگیرد.

## ۸. CI، mirror و GC را فعال کنید

برای CI فقط مسیر webhook را پشت HTTPS منتشر کنید. باید TLS ورودی، تأیید TLS
backend و peerهای مجاز ingress در NetworkPolicy تنظیم شوند. توکن webhook را
به‌صورت masked/protected در CI ذخیره کنید. نمونه درخواست CI در راهنمای انگلیسی
بخش ۸ آمده است؛ از قراردادن token در repository خودداری کنید.

mirror نیازمند config_path صحیح containerd است. GC فعلاً استفاده از دیسک `/`
را از node-exporter می‌خواند؛ اگر imageها روی دیسک جدا هستند آن را فعال نکنید.
پاک‌سازی kubelet را خاموش نکنید. انتقال snapshot به‌صورت پیش‌فرض بسته است؛
برای رفع خطای نبود blob، ریسک snapshot غیرقابل‌اعتبارسنجی را خودکار نپذیرید.

## ۹. خطاها و بازگشت به نسخه قبل

| خطا | کار لازم |
|---|---|
| FailedMount | وجود Secretها و مسیرهای نود را در describe pod بررسی کنید. |
| ImagePullBackOff خود worker | image و digest، دسترسی registry و pull Secret را بررسی کنید. |
| CrashLoopBackOff | logs را بخوانید؛ نبود TLS یا توکن عمداً startup را متوقف می‌کند. |
| x509 | CA، SAN، تاریخ انقضا و ساعت سیستم را اصلاح کنید؛ verification را خاموش نکنید. |
| webhook با 401 | توکن مخصوص CI را بدهید. |
| webhook با 503 | آماده‌بودن workerها، policy شبکه و فضای دیسک را بررسی کنید. |

قبل از ارتقا:

```bash
helm -n adhdduck history adhdduck
helm -n adhdduck get values adhdduck > .local/previous-values.yaml
```

برای rollback، شماره revision سالم قبلی را در دستور
`helm -n adhdduck rollback adhdduck REVISION --wait --timeout 5m` جایگزین کنید.
artifactها و Secretهای سازگار با آن نسخه باید موجود باشند. برای حذف، ابتدا
mirror را خاموش کنید، پاک‌شدن فایل‌های متعلق به پروژه را روی همه نودها بررسی
کنید و بعد Helm uninstall بزنید. پوشه داده containerd را حذف نکنید.

تست‌های سایت شما شامل قطعی registry، restart، قطع انتقال، فشار دیسک، rotation
و rollback است؛ جدول نتایج قابل ثبت در [production.md](production.md) قرار دارد.
