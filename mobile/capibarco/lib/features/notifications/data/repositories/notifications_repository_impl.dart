import 'dart:convert';

import '../../../../core/cache/json_cache_store.dart';
import '../../../../core/error/error_mapper.dart';
import '../../domain/entities/notification_item.dart';
import '../datasources/notifications_remote_data_source.dart';
import '../dtos/notification_dto.dart';

class NotificationsRepositoryImpl {
  const NotificationsRepositoryImpl({
    required NotificationsRemoteDataSource remoteDataSource,
    required JsonCacheStore cacheStore,
    required ErrorMapper errorMapper,
  }) : _remoteDataSource = remoteDataSource,
       _cacheStore = cacheStore,
       _errorMapper = errorMapper;

  final NotificationsRemoteDataSource _remoteDataSource;
  final JsonCacheStore _cacheStore;
  final ErrorMapper _errorMapper;

  Future<NotificationsPageEntity> listNotifications({
    required String cacheScope,
  }) async {
    final cacheKey = 'notifications:$cacheScope';
    try {
      final remote = await _remoteDataSource.listNotifications();
      await _cacheStore.write(
        cacheKey,
        jsonEncode(<String, dynamic>{
          'notifications': remote.items
              .map(
                (item) => <String, dynamic>{
                  'notification_id': item.id,
                  'title': item.title,
                  'body': item.body,
                  'type': item.type,
                  'status': item.status,
                  'data': item.data,
                  'created_at': item.createdAt.toIso8601String(),
                  'read_at': item.readAt?.toIso8601String(),
                },
              )
              .toList(),
          'page': <String, dynamic>{'next_page_token': remote.nextPageToken},
        }),
      );
      return remote.toDomain();
    } catch (error) {
      final mappedError = _errorMapper.map(error);
      final cachedRaw = _cacheStore.read(cacheKey);
      if (cachedRaw != null && mappedError.isRetryable) {
        final cached = NotificationsPageDto.fromJson(
          jsonDecode(cachedRaw) as Map<String, dynamic>,
          isStale: true,
        );
        return cached.toDomain();
      }
      throw mappedError;
    }
  }

  Future<void> markAsRead(String notificationId) async {
    await _remoteDataSource.markAsRead(notificationId);
  }
}
