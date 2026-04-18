import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../../../bootstrap/providers.dart';
import '../../../core/config/environment.dart';
import '../../../core/network/network_providers.dart';
import '../../../core/network/rest_service_client.dart';
import '../data/api/notifications_api_client.dart';
import '../data/datasources/notifications_remote_data_source.dart';
import '../data/repositories/notifications_repository_impl.dart';
import '../domain/entities/notification_item.dart';
import '../../auth/presentation/auth_controller.dart';

class NotificationsState {
  const NotificationsState({
    this.items = const <NotificationItemEntity>[],
    this.isLoading = false,
    this.errorMessage,
    this.isStale = false,
  });

  final List<NotificationItemEntity> items;
  final bool isLoading;
  final String? errorMessage;
  final bool isStale;

  NotificationsState copyWith({
    List<NotificationItemEntity>? items,
    bool? isLoading,
    String? errorMessage,
    bool clearError = false,
    bool? isStale,
  }) {
    return NotificationsState(
      items: items ?? this.items,
      isLoading: isLoading ?? this.isLoading,
      errorMessage: clearError ? null : (errorMessage ?? this.errorMessage),
      isStale: isStale ?? this.isStale,
    );
  }
}

final notificationsRepositoryProvider = Provider<NotificationsRepositoryImpl>((
  ref,
) {
  final environment = ref.watch(appEnvironmentProvider);
  return NotificationsRepositoryImpl(
    remoteDataSource: NotificationsRemoteDataSource(
      NotificationsApiClient(
        RestServiceClient(
          dio: ref.watch(authenticatedDioProvider),
          config: environment.service(ServiceKind.notifications),
        ),
      ),
    ),
    cacheStore: ref.watch(cacheStoreProvider),
    errorMapper: ref.watch(errorMapperProvider),
  );
});

final notificationsControllerProvider =
    NotifierProvider<NotificationsController, NotificationsState>(
      NotificationsController.new,
    );

class NotificationsController extends Notifier<NotificationsState> {
  NotificationsRepositoryImpl get _repository =>
      ref.read(notificationsRepositoryProvider);

  @override
  NotificationsState build() => const NotificationsState();

  Future<void> load() async {
    state = state.copyWith(isLoading: true, clearError: true);
    try {
      final page = await _repository.listNotifications();
      state = state.copyWith(
        items: page.items,
        isLoading: false,
        isStale: page.isStale,
      );
    } catch (error) {
      state = state.copyWith(isLoading: false, errorMessage: error.toString());
    }
  }

  Future<void> markAsRead(NotificationItemEntity item) async {
    await _repository.markAsRead(item.id);
    state = state.copyWith(
      items: state.items
          .map(
            (existing) => existing.id == item.id
                ? NotificationItemEntity(
                    id: existing.id,
                    title: existing.title,
                    body: existing.body,
                    type: existing.type,
                    status: 'read',
                    createdAt: existing.createdAt,
                    readAt: DateTime.now().toUtc(),
                  )
                : existing,
          )
          .toList(),
    );
  }
}
