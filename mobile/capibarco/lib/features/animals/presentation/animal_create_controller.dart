import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:image_picker/image_picker.dart';

import '../../../bootstrap/providers.dart';
import '../../../core/config/environment.dart';
import '../../../core/network/network_providers.dart';
import '../../../core/network/rest_service_client.dart';
import '../../auth/presentation/auth_controller.dart';
import '../../profile/presentation/profile_controller.dart';
import '../data/api/animals_api_client.dart';
import '../data/datasources/animals_remote_data_source.dart';
import '../data/repositories/animals_repository_impl.dart';

class AnimalCreateState {
  const AnimalCreateState({
    this.isSubmitting = false,
    this.errorMessage,
    this.successMessage,
  });

  final bool isSubmitting;
  final String? errorMessage;
  final String? successMessage;

  AnimalCreateState copyWith({
    bool? isSubmitting,
    String? errorMessage,
    bool clearError = false,
    String? successMessage,
    bool clearSuccess = false,
  }) {
    return AnimalCreateState(
      isSubmitting: isSubmitting ?? this.isSubmitting,
      errorMessage: clearError ? null : (errorMessage ?? this.errorMessage),
      successMessage: clearSuccess
          ? null
          : (successMessage ?? this.successMessage),
    );
  }
}

final animalsRepositoryProvider = Provider<AnimalsRepositoryImpl>((ref) {
  final environment = ref.watch(appEnvironmentProvider);
  return AnimalsRepositoryImpl(
    remoteDataSource: AnimalsRemoteDataSource(
      AnimalsApiClient(
        RestServiceClient(
          dio: ref.watch(authenticatedDioProvider),
          config: environment.service(ServiceKind.animals),
        ),
      ),
    ),
    errorMapper: ref.watch(errorMapperProvider),
  );
});

final animalCreateControllerProvider =
    NotifierProvider<AnimalCreateController, AnimalCreateState>(
      AnimalCreateController.new,
    );

class AnimalCreateController extends Notifier<AnimalCreateState> {
  AnimalsRepositoryImpl get _repository => ref.read(animalsRepositoryProvider);

  @override
  AnimalCreateState build() => const AnimalCreateState();

  Future<bool> createAnimal({
    required String name,
    required String species,
    required String breed,
    required String sex,
    required String size,
    required int ageMonths,
    required String description,
    required List<String> traits,
    required bool vaccinated,
    required bool sterilized,
    required bool publishNow,
    XFile? photo,
  }) async {
    final profile = ref.read(profileControllerProvider).profile;
    if (profile == null) {
      state = state.copyWith(
        errorMessage: 'Create or update your profile first.',
      );
      return false;
    }

    state = state.copyWith(
      isSubmitting: true,
      clearError: true,
      clearSuccess: true,
    );

    try {
      final animal = await _repository.createAnimal(
        ownerProfileId: profile.id,
        ownerType: _mapOwnerType(profile.typeCode),
        name: name,
        species: species,
        breed: breed,
        sex: sex,
        size: size,
        ageMonths: ageMonths,
        description: description,
        traits: traits,
        vaccinated: vaccinated,
        sterilized: sterilized,
        publishNow: publishNow,
        city: profile.city,
      );
      if (photo != null) {
        await _repository.uploadAnimalPhoto(
          animalId: animal.id,
          photoPath: photo.path,
          fileName: photo.name,
        );
      }
      state = state.copyWith(
        isSubmitting: false,
        successMessage: photo == null
            ? 'Pet profile created.'
            : 'Pet profile created with photo.',
      );
      return true;
    } catch (error) {
      state = state.copyWith(
        isSubmitting: false,
        errorMessage: error.toString(),
      );
      return false;
    }
  }

  String _mapOwnerType(String profileType) {
    switch (profileType) {
      case 'PROFILE_TYPE_KENNEL':
        return 'OWNER_TYPE_KENNEL';
      case 'PROFILE_TYPE_SHELTER':
        return 'OWNER_TYPE_SHELTER';
      default:
        return 'OWNER_TYPE_USER';
    }
  }
}
