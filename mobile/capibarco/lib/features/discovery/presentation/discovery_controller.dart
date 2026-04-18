import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../../../bootstrap/providers.dart';
import '../../../core/config/environment.dart';
import '../../../core/network/network_providers.dart';
import '../../../core/network/rest_service_client.dart';
import '../../../features/auth/presentation/auth_controller.dart';
import '../data/api/discovery_api_client.dart';
import '../data/datasources/discovery_remote_data_source.dart';
import '../data/repositories/discovery_repository_impl.dart';
import '../domain/entities/profile_summary.dart';

class DiscoveryState {
  const DiscoveryState({
    this.items = const <ProfileSummaryEntity>[],
    this.query = '',
    this.city = '',
    this.nextPageToken = '',
    this.isLoading = false,
    this.errorMessage,
    this.isStale = false,
  });

  final List<ProfileSummaryEntity> items;
  final String query;
  final String city;
  final String nextPageToken;
  final bool isLoading;
  final String? errorMessage;
  final bool isStale;

  DiscoveryState copyWith({
    List<ProfileSummaryEntity>? items,
    String? query,
    String? city,
    String? nextPageToken,
    bool? isLoading,
    String? errorMessage,
    bool clearError = false,
    bool? isStale,
  }) {
    return DiscoveryState(
      items: items ?? this.items,
      query: query ?? this.query,
      city: city ?? this.city,
      nextPageToken: nextPageToken ?? this.nextPageToken,
      isLoading: isLoading ?? this.isLoading,
      errorMessage: clearError ? null : (errorMessage ?? this.errorMessage),
      isStale: isStale ?? this.isStale,
    );
  }
}

final discoveryRepositoryProvider = Provider<DiscoveryRepositoryImpl>((ref) {
  final environment = ref.watch(appEnvironmentProvider);
  return DiscoveryRepositoryImpl(
    remoteDataSource: DiscoveryRemoteDataSource(
      DiscoveryApiClient(
        RestServiceClient(
          dio: ref.watch(authenticatedDioProvider),
          config: environment.service(ServiceKind.profiles),
        ),
      ),
    ),
    cacheStore: ref.watch(cacheStoreProvider),
    errorMapper: ref.watch(errorMapperProvider),
  );
});

final discoveryControllerProvider =
    NotifierProvider<DiscoveryController, DiscoveryState>(
      DiscoveryController.new,
    );

class DiscoveryController extends Notifier<DiscoveryState> {
  DiscoveryRepositoryImpl get _repository =>
      ref.read(discoveryRepositoryProvider);

  @override
  DiscoveryState build() => const DiscoveryState();

  Future<void> search({required String query, required String city}) async {
    state = state.copyWith(
      query: query,
      city: city,
      isLoading: true,
      clearError: true,
    );
    try {
      final page = await _repository.searchProfiles(query: query, city: city);
      state = state.copyWith(
        items: page.items,
        nextPageToken: page.nextPageToken,
        isLoading: false,
        isStale: page.isStale,
      );
    } catch (error) {
      state = state.copyWith(isLoading: false, errorMessage: error.toString());
    }
  }
}
