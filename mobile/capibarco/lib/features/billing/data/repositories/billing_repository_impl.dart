import '../../../../core/error/error_mapper.dart';
import '../../domain/entities/donation_intent.dart';
import '../datasources/billing_remote_data_source.dart';

class BillingRepositoryImpl {
  const BillingRepositoryImpl({
    required BillingRemoteDataSource remoteDataSource,
    required ErrorMapper errorMapper,
  }) : _remoteDataSource = remoteDataSource,
       _errorMapper = errorMapper;

  final BillingRemoteDataSource _remoteDataSource;
  final ErrorMapper _errorMapper;

  Future<DonationIntentEntity> createAnimalDonation({
    required String animalId,
    required int amountRubles,
  }) async {
    try {
      final response = await _remoteDataSource.createDonationIntent(
        targetType: 'DONATION_TARGET_TYPE_ANIMAL',
        targetId: animalId,
        units: amountRubles,
        nanos: 0,
        currencyCode: 'RUB',
      );
      return response.toDomain();
    } catch (error) {
      throw _errorMapper.map(error);
    }
  }
}
