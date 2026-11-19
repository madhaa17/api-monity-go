# Setup Env

Yang perlu diubah setelah auth pindah ke cookie lewat proxy web.

Buat satu token, lalu pakai di web **dan** backend (nilainya harus sama):

```bash
openssl rand -hex 32
```

---

## Lokal

### `web/.env.local`

| Aksi | Env | Nilai |
|---|---|---|
| ❌ Hapus | `NEXT_PUBLIC_API_URL` | |
| ➕ Tambah | `API_URL` | `http://localhost:8080` |

Hasil akhirnya:

```env
API_URL=http://localhost:8080
```

### `backend/.env`

| Aksi | Env | Nilai |
|---|---|---|
| ✏️ Ubah | `CORS_ALLOWED_ORIGINS` | kosongkan |

Hasil akhirnya:

```env
CORS_ALLOWED_ORIGINS=
```

`INTERNAL_API_TOKEN` tidak perlu diisi di lokal.

Setelah itu restart backend dan `npm run dev`.

---

## Production

Kerjakan **berurutan**.

### Langkah 1: Vercel (web)

| Aksi | Env | Nilai |
|---|---|---|
| ➕ Tambah | `API_URL` | `https://api-assetmonitor.my.id` |
| ➕ Tambah | `INTERNAL_API_TOKEN` | token tadi |

Centang Production dan Preview, lalu **Redeploy**.

### Langkah 2: Dokploy (backend)

| Aksi | Env | Nilai |
|---|---|---|
| ➕ Tambah | `INTERNAL_API_TOKEN` | token yang sama |
| ✏️ Ubah | `CORS_ALLOWED_ORIGINS` | kosongkan |

Tulis tanpa tanda kutip, lalu klik **Deploy** (bukan hanya Save).

### Langkah 3: Vercel (web)

| Aksi | Env |
|---|---|
| ❌ Hapus | `NEXT_PUBLIC_API_URL` |

Lalu **Redeploy**.

---

## Kalau error

| Masalah | Solusi |
|---|---|
| Semua request 403 | Token di Vercel dan Dokploy beda, atau belum redeploy |
| Backend tidak mau start | `INTERNAL_API_TOKEN` belum diisi / kurang dari 32 karakter |
| Env di Dokploy tidak terbaca | Belum klik Deploy |
