import 'dart:typed_data';

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

  Future<bool> saveDraft({
    String? animalId,
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
    XFile? photo,
    Uint8List? photoBytes,
    String? createIdempotencyKey,
  }) => _submitAnimal(
    animalId: animalId,
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
    photo: photo,
    photoBytes: photoBytes,
    createIdempotencyKey: createIdempotencyKey,
    publish: false,
  );

  Future<bool> publishAnimal({
    String? animalId,
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
    XFile? photo,
    Uint8List? photoBytes,
    bool hasExistingPhoto = false,
    String? createIdempotencyKey,
  }) {
    if (photo == null && !hasExistingPhoto) {
      state = state.copyWith(
        errorMessage: 'Add at least one photo before publishing.',
        clearSuccess: true,
      );
      return Future<bool>.value(false);
    }
    return _submitAnimal(
      animalId: animalId,
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
      photo: photo,
      photoBytes: photoBytes,
      createIdempotencyKey: createIdempotencyKey,
      publish: true,
    );
  }

  Future<bool> _submitAnimal({
    String? animalId,
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
    XFile? photo,
    Uint8List? photoBytes,
    String? createIdempotencyKey,
    required bool publish,
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
      final targetAnimalId = animalId == null || animalId.isEmpty
          ? (await _repository.createAnimalDraft(
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
              city: profile.city,
              idempotencyKey: createIdempotencyKey,
            )).id
          : (await _repository.updateAnimalDraft(
              animalId: animalId,
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
              city: profile.city,
            )).id;
      if (photo != null) {
        final bytes = photoBytes ?? await photo.readAsBytes();
        await _repository.uploadAnimalPhoto(
          animalId: targetAnimalId,
          photoBytes: bytes,
          fileName: photo.name,
        );
      }
      if (publish) {
        await _repository.publishAnimal(animalId: targetAnimalId);
      }
      state = state.copyWith(
        isSubmitting: false,
        successMessage: publish ? 'Pet profile published.' : 'Pet draft saved.',
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
