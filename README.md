# GTK4 Go Uygulaması - GNOME Tema Desteği

Bu uygulama, GNOME'un karanlık/açık tema tercihini otomatik olarak destekleyen GTK4 Go uygulamasıdır. Uygulama, modern bir arayüz ile HeaderBar, ListBox ve menü desteği içerir.

## Özellikler

- ✅ GNOME'un sistem tema tercihini otomatik algılama
- ✅ `org.gnome.desktop.interface color-scheme` ayarını okuma
- ✅ Tema değişikliklerini gerçek zamanlı olarak takip etme
- ✅ GTK4'ün `gtk-application-prefer-dark-theme` özelliğini kullanma

## Nasıl Çalışır

Uygulama aşağıdaki yöntemlerle GNOME tema desteği sağlar:

1. **GSettings Entegrasyonu**: `org.gnome.desktop.interface` şemasından `color-scheme` ayarını okur
2. **Gerçek Zamanlı Takip**: Sistem tema değişikliklerini dinler ve otomatik olarak günceller
3. **GTK4 Uyumluluğu**: `gtk-application-prefer-dark-theme` özelliğini kullanarak GTK4'ün yerli tema desteğinden yararlanır

## Derleme ve Çalıştırma

```bash
# Bağımlılıkları yükle
go mod tidy

# Derle
go build -o go-gtk4-app .

# Çalıştır
./go-gtk4-app
```

## Tema Testi

Uygulamanın tema desteğini test etmek için otomatik test scripti kullanabilirsiniz:

```bash
# Otomatik tema testi
./test-theme.sh
```

Veya manuel olarak:

```bash
# Karanlık temaya geç
gsettings set org.gnome.desktop.interface color-scheme 'prefer-dark'

# Açık temaya geç
gsettings set org.gnome.desktop.interface color-scheme 'default'

# Mevcut ayarı kontrol et
gsettings get org.gnome.desktop.interface color-scheme
```

## Teknik Detaylar

### Kullanılan Kütüphaneler
- `github.com/diamondburned/gotk4/pkg/gtk/v4` - GTK4 bağlamaları
- `github.com/diamondburned/gotk4/pkg/gio/v2` - GIO/GSettings desteği

### Tema Algılama Mantığı
1. Uygulama başladığında `org.gnome.desktop.interface` şemasından `color-scheme` değeri okunur
2. Bu değer `prefer-dark` ise `gtk-application-prefer-dark-theme` true olarak ayarlanır
3. GSettings değişiklik sinyalleri dinlenerek tema değişiklikleri gerçek zamanlı olarak takip edilir

### Fallback Mekanizması
GNOME ayarları mevcut değilse, uygulama GTK tema adından çıkarım yapar:
- `Adwaita-dark` → Karanlık tema
- `Adwaita` → Açık tema

## Sorun Giderme

### GNOME Ayarları Bulunamıyor
Eğer `org.gnome.desktop.interface` şeması bulunamazsa:
```bash
# Şemaların listesini kontrol et
gsettings list-schemas | grep desktop.interface

# GNOME ayarlarını sıfırla
gsettings reset org.gnome.desktop.interface color-scheme
```

### GTK Tema Ayarları
Manuel GTK tema ayarları için:
```bash
# GTK ayarlarını kontrol et
gsettings list-keys org.gnome.desktop.interface

# Tema adını kontrol et
gsettings get org.gnome.desktop.interface gtk-theme
```

## Lisans

Bu proje MIT lisansı altında lisanslanmıştır.
