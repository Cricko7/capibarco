abstract interface class JsonCacheStore {
  String? read(String key);

  Future<void> write(String key, String value);

  Future<void> remove(String key);
}
