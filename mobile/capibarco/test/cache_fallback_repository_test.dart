import 'dart:io';

import 'package:capibarco/core/cache/json_cache_store.dart';
import 'package:capibarco/core/config/environment.dart';
import 'package:capibarco/core/error/app_exception.dart';
import 'package:capibarco/core/error/error_mapper.dart';
import 'package:capibarco/core/network/rest_service_client.dart';
import 'package:capibarco/features/feed/data/api/feed_api_client.dart';
import 'package:capibarco/features/feed/data/datasources/feed_remote_data_source.dart';
import 'package:capibarco/features/feed/data/repositories/feed_repository_impl.dart';
import 'package:capibarco/features/notifications/data/api/notifications_api_client.dart';
import 'package:capibarco/features/notifications/data/datasources/notifications_remote_data_source.dart';
import 'package:capibarco/features/notifications/data/repositories/notifications_repository_impl.dart';
import 'package:dio/dio.dart';
import 'package:flutter_test/flutter_test.dart';

class _MemoryCacheStore implements JsonCacheStore {
  final Map<String, String> _store = <String, String>{};

  @override
  String? read(String key) => _store[key];

  @override
  Future<void> remove(String key) async {
    _store.remove(key);
  }

  @override
  Future<void> write(String key, String value) async {
    _store[key] = value;
  }
}

class _FailingInterceptor extends Interceptor {
  _FailingInterceptor(this.errorFactory);

  final DioException Function(RequestOptions options) errorFactory;

  @override
  void onRequest(RequestOptions options, RequestInterceptorHandler handler) {
    handler.reject(errorFactory(options));
  }
}

RestServiceClient _restClient(
  DioException Function(RequestOptions options) errorFactory,
) {
  final dio = Dio(BaseOptions(baseUrl: 'https://example.test'));
  dio.interceptors.add(_FailingInterceptor(errorFactory));
  return RestServiceClient(
    dio: dio,
    config: const ServiceConfig(
      baseUrl: 'https://example.test',
      apiVersion: 'v1',
      protocol: TransportProtocol.rest,
    ),
  );
}

void main() {
  group('FeedRepositoryImpl cache fallback', () {
    test('returns stale cached feed only for retryable errors', () async {
      final cacheStore = _MemoryCacheStore();
      await cacheStore.write('feed:', '''
        {
          "cards": [
            {
              "feed_card_id": "card-1",
              "feed_session_id": "session-1",
              "animal": {
                "animal_id": "animal-1",
                "owner_profile_id": "owner-1",
                "name": "Mila",
                "species": "SPECIES_CAT",
                "description": "Friendly",
                "photos": [{"url": "https://example.test/cat.jpg"}],
                "location": {"city": "Moscow"}
              },
              "owner_display_name": "Shelter",
              "boosted": false,
              "ranking_reasons": []
            }
          ],
          "next_page_token": "",
          "feed_session_id": "session-1"
        }
        ''');

      final repository = FeedRepositoryImpl(
        remoteDataSource: FeedRemoteDataSource(
          FeedApiClient(
            _restClient(
              (options) => DioException(
                requestOptions: options,
                type: DioExceptionType.connectionError,
                error: const SocketException('offline'),
              ),
            ),
          ),
        ),
        cacheStore: cacheStore,
        errorMapper: const ErrorMapper(),
      );

      final page = await repository.getFeed();
      expect(page.isStale, isTrue);
      expect(page.cards, hasLength(1));
      expect(page.cards.first.name, 'Mila');
    });

    test('surfaces non-retryable feed errors instead of stale cache', () async {
      final cacheStore = _MemoryCacheStore();
      await cacheStore.write(
        'feed:',
        '{"cards":[],"next_page_token":"","feed_session_id":""}',
      );

      final repository = FeedRepositoryImpl(
        remoteDataSource: FeedRemoteDataSource(
          FeedApiClient(
            _restClient(
              (options) => DioException(
                requestOptions: options,
                response: Response<dynamic>(
                  requestOptions: options,
                  statusCode: 401,
                  data: <String, dynamic>{'detail': 'Unauthorized'},
                ),
                type: DioExceptionType.badResponse,
              ),
            ),
          ),
        ),
        cacheStore: cacheStore,
        errorMapper: const ErrorMapper(),
      );

      await expectLater(
        repository.getFeed(),
        throwsA(
          isA<AppException>().having(
            (error) => error.isUnauthorized,
            'isUnauthorized',
            isTrue,
          ),
        ),
      );
    });
  });

  group('NotificationsRepositoryImpl cache fallback', () {
    test(
      'surfaces non-retryable notification errors instead of stale cache',
      () async {
        final cacheStore = _MemoryCacheStore();
        await cacheStore.write('notifications', '''
          {
            "notifications": [
              {
                "notification_id": "notification-1",
                "title": "Hello",
                "body": "World",
                "type": "NOTIFICATION_TYPE_SYSTEM",
                "status": "NOTIFICATION_STATUS_UNREAD",
                "created_at": "2026-04-18T10:00:00Z"
              }
            ],
            "page": {"next_page_token": ""}
          }
          ''');

        final repository = NotificationsRepositoryImpl(
          remoteDataSource: NotificationsRemoteDataSource(
            NotificationsApiClient(
              _restClient(
                (options) => DioException(
                  requestOptions: options,
                  response: Response<dynamic>(
                    requestOptions: options,
                    statusCode: 500,
                    data: <String, dynamic>{'detail': 'Internal error'},
                  ),
                  type: DioExceptionType.badResponse,
                ),
              ),
            ),
          ),
          cacheStore: cacheStore,
          errorMapper: const ErrorMapper(),
        );

        await expectLater(
          repository.listNotifications(),
          throwsA(
            isA<AppException>().having(
              (error) => error.statusCode,
              'statusCode',
              500,
            ),
          ),
        );
      },
    );
  });
}
