import '../../../../core/error/error_mapper.dart';
import '../../domain/entities/animal_listing.dart';
import '../datasources/animals_remote_data_source.dart';

class AnimalsRepositoryImpl {
  const AnimalsRepositoryImpl({
    required AnimalsRemoteDataSource remoteDataSource,
    required ErrorMapper errorMapper,
  }) : _remoteDataSource = remoteDataSource,
       _errorMapper = errorMapper;

  final AnimalsRemoteDataSource _remoteDataSource;
  final ErrorMapper _errorMapper;

  Future<AnimalListingEntity> createAnimal({
    required String ownerProfileId,
    required String ownerType,
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
    required String city,
  }) async {
    try {
      final animal = await _remoteDataSource.createAnimal(
        ownerProfileId: ownerProfileId,
        ownerType: ownerType,
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
        status: publishNow ? 'ANIMAL_STATUS_AVAILABLE' : 'ANIMAL_STATUS_DRAFT',
        visibility: publishNow ? 'VISIBILITY_PUBLIC' : 'VISIBILITY_PRIVATE',
        city: city,
      );
      return animal.toDomain();
    } catch (error) {
      throw _errorMapper.map(error);
    }
  }

  Future<AnimalListingEntity> uploadAnimalPhoto({
    required String animalId,
    required String photoPath,
    required String fileName,
  }) async {
    try {
      final animal = await _remoteDataSource.uploadAnimalPhoto(
        animalId: animalId,
        photoPath: photoPath,
        fileName: fileName,
      );
      return animal.toDomain();
    } catch (error) {
      throw _errorMapper.map(error);
    }
  }
}
