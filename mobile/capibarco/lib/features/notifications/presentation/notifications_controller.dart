import 'package:flutter/foundation.dart';
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
    this.profileId = '',
    this.items = const <NotificationItemEntity>[],
    this.isLoading = false,
    this.hasLoaded = false,
    this.errorMessage,
    this.isStale = false,
  });

  final String profileId;
  final List<NotificationItemEntity> items;
  final bool isLoading;
  final bool hasLoaded;
  final String? errorMessage;
  final bool isStale;

  NotificationsState copyWith({
    String? profileId,
    List<NotificationItemEntity>? items,
    bool? isLoading,
    bool? hasLoaded,
    String? errorMessage,
    bool clearError = false,
    bool? isStale,
  }) {
    return NotificationsState(
      profileId: profileId ?? this.profileId,
      items: items ?? this.items,
      isLoading: isLoading ?? this.isLoading,
      hasLoaded: hasLoaded ?? this.hasLoaded,
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

  int _requestSerial = 0;

  @override
  NotificationsState build() {
    final profileId = ref.watch(
      authControllerProvider.select((auth) => auth.session?.user.id ?? ''),
    );
    return NotificationsState(profileId: profileId);
  }

  Future<void> load({bool background = false}) async {
    final profileId =
        ref.read(authControllerProvider.notifier).currentProfileId ?? '';

    if (profileId.isEmpty) {
      state = const NotificationsState();
      return;
    }

    final switchedProfile = state.profileId != profileId;
    if (background && state.isLoading && !switchedProfile) {
      return;
    }
    final requestId = ++_requestSerial;
    if (!background || switchedProfile) {
      state = NotificationsState(
        profileId: profileId,
        items: switchedProfile ? const <NotificationItemEntity>[] : state.items,
        isLoading: true,
        hasLoaded: switchedProfile ? false : state.hasLoaded,
        isStale: switchedProfile ? false : state.isStale,
      );
    }
    try {
      final page = await _repository.listNotifications(cacheScope: profileId);
      final currentProfileId =
          ref.read(authControllerProvider.notifier).currentProfileId ?? '';
      if (requestId != _requestSerial || currentProfileId != profileId) {
        return;
      }
      final visibleItems = page.items.where(_shouldShowInInbox).toList();
      if (kDebugMode) {
        debugPrint(
          '[Notifications] loaded ${visibleItems.length}/${page.items.length} item(s), stale=${page.isStale}',
        );
      }
      state = state.copyWith(
        profileId: profileId,
        items: visibleItems,
        isLoading: false,
        hasLoaded: true,
        isStale: page.isStale,
        clearError: true,
      );
    } catch (error) {
      final currentProfileId =
          ref.read(authControllerProvider.notifier).currentProfileId ?? '';
      if (requestId != _requestSerial || currentProfileId != profileId) {
        return;
      }
      if (kDebugMode) {
        debugPrint('[Notifications] load failed: $error');
      }
      if (background && state.items.isNotEmpty) {
        return;
      }
      state = state.copyWith(
        profileId: profileId,
        isLoading: false,
        hasLoaded: true,
        errorMessage: error.toString(),
      );
    }
  }

  Future<void> refreshRealtime() => load(background: true);

  Future<void> markAsRead(NotificationItemEntity item) async {
    await _repository.markAsRead(item.id);
    state = state.copyWith(
      items: state.items.where((existing) => existing.id != item.id).toList(),
    );
  }

  bool _shouldShowInInbox(NotificationItemEntity item) {
    return item.status != 'read' && item.readAt == null;
  }
}
