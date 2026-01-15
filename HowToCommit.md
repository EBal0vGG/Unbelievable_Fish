
## 1. Добавляешь файл и коммитишь в свою ветку

```bash
git branch # проверяем, что мы в своей ветке
git add myfile.txt  # или git add .
git commit -m "Add myfile"
```

Теперь:

* файл **есть в твоей ветке**
* в `main` его **ещё нет**

---

## 2. Мержишь `dev` (свою ветку) → `main`, если это стабильная функция, которая не вызовет конфликтов

Переключаешься в `main`:

```bash
git switch main
```

Делаешь merge:

```bash
git merge dev
```

Теперь:

* файл **есть и в `main`, и в `dev`**
* история чистая, без конфликтов

---

## 3. Пуш

```bash
git push origin dev
git push origin main
```

или один раз с `-u`, если ветки новые:

```bash
git push -u origin dev
git push -u origin main
```

