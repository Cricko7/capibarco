import 'package:shared_preferences/shared_preferences.dart';

import 'json_cache_store.dart';

class SharedPreferencesCacheStore implements JsonCacheStore {
  const SharedPreferencesCacheStore(this._sharedPreferences);

  final SharedPreferences _sharedPreferences;

  @override
  String? read(String key) => _sharedPreferences.getString(key);

  @override
  Future<void> remove(String key) => _sharedPreferences.remove(key);

  @override
  Future<void> write(String key, String value) =>
      _sharedPreferences.setString(key, value);
}
