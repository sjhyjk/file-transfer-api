-- 既存のtagsカラムに対してGINインデックスを作成
CREATE INDEX IF NOT EXISTS idx_file_metadata_tags ON file_metadata USING GIN (tags);
```

> **なぜ GIN なのか？**
> PostgreSQLの配列型に対して、`@>`（包含）などの配列演算子を用いる場合、通常の B-tree インデックスでは効果がありません。**GIN (Generalized Inverted Index)** を使うことで、タグがどれだけ増えても、特定のタグを含むレコードを爆速で見つけられるようになります。数学を専攻されていた永田さんなら、「集合の包含関係を転置インデックスで管理する」という効率の良さがしっくりくるはずです。

---

### 2. コード側の確認

永田さんの実装ですでに配列演算子（`@>`）を使っていれば、SQLを流すだけで自動的にこのインデックスが使われるようになります。もし `WHERE tags LIKE ...` のような文字列検索にしている場合は、インデックスを活かすために以下の形式のSQLを発行するようにしてください。

```sql
-- 内部的なクエリイメージ（pgx等で発行するもの）
-- SELECT * FROM files WHERE tags @> ARRAY['重要'];
