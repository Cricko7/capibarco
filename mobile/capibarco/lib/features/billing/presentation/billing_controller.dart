import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../../../bootstrap/providers.dart';
import '../../../core/config/environment.dart';
import '../../../core/network/network_providers.dart';
import '../../../core/network/rest_service_client.dart';
import '../../auth/presentation/auth_controller.dart';
import '../data/api/billing_api_client.dart';
import '../data/datasources/billing_remote_data_source.dart';
import '../data/repositories/billing_repository_impl.dart';
import '../domain/entities/donation_intent.dart';

class BillingState {
  const BillingState({
    this.isSubmitting = false,
    this.errorMessage,
    this.intent,
  });

  final bool isSubmitting;
  final String? errorMessage;
  final DonationIntentEntity? intent;

  BillingState copyWith({
    bool? isSubmitting,
    String? errorMessage,
    bool clearError = false,
    DonationIntentEntity? intent,
    bool clearIntent = false,
  }) {
    return BillingState(
      isSubmitting: isSubmitting ?? this.isSubmitting,
      errorMessage: clearError ? null : (errorMessage ?? this.errorMessage),
      intent: clearIntent ? null : (intent ?? this.intent),
    );
  }
}

final billingRepositoryProvider = Provider<BillingRepositoryImpl>((ref) {
  final environment = ref.watch(appEnvironmentProvider);
  return BillingRepositoryImpl(
    remoteDataSource: BillingRemoteDataSource(
      BillingApiClient(
        RestServiceClient(
          dio: ref.watch(authenticatedDioProvider),
          config: environment.service(ServiceKind.billing),
        ),
      ),
    ),
    errorMapper: ref.watch(errorMapperProvider),
  );
});

final billingControllerProvider =
    NotifierProvider<BillingController, BillingState>(BillingController.new);

class BillingController extends Notifier<BillingState> {
  BillingRepositoryImpl get _repository => ref.read(billingRepositoryProvider);

  @override
  BillingState build() => const BillingState();

  void reset() {
    state = const BillingState();
  }

  Future<bool> createAnimalDonation({
    required String animalId,
    required int amountRubles,
  }) async {
    if (amountRubles <= 0) {
      state = state.copyWith(
        errorMessage: 'Enter an amount greater than zero.',
      );
      return false;
    }

    state = state.copyWith(
      isSubmitting: true,
      clearError: true,
      clearIntent: true,
    );

    try {
      final intent = await _repository.createAnimalDonation(
        animalId: animalId,
        amountRubles: amountRubles,
      );
      state = state.copyWith(isSubmitting: false, intent: intent);
      return true;
    } catch (error) {
      state = state.copyWith(
        isSubmitting: false,
        errorMessage: error.toString(),
      );
      return false;
    }
  }
}
