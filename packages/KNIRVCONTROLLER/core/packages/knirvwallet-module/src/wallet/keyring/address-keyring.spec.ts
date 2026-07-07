import { strToSignedTx, txToDocument } from '../../utils';
import { AddressKeyring } from './address-keyring';

describe('create address keyring', () => {
  it('create address keyring from address', async () => {
    const address = 'g1jg8mtutu9khhfwc4nxmuhcpftf0pajdhfvsqf5';
    const addressKeyring = await AddressKeyring.fromAddress(address);

    expect(addressKeyring.id).not.toBeNull();
    expect([...addressKeyring.addressBytes]).toStrictEqual([
      146, 15, 181, 241, 124, 45, 175, 116, 187, 21, 153, 183, 203, 224, 41, 90, 94, 30, 201, 183,
    ]);
    expect(addressKeyring.type.toString()).toBe('ADDRESS');
  });
});

describe('tx encode of address keyring', () => {
  it('str to signedTx', async () => {
    const str =
      '{"msg":[{"@type":"/vm.m_call","caller":"g1jg8mtutu9khhfwc4nxmuhcpftf0pajdhfvsqf5","send":"","pkg_path":"gno.land/r/demo/tong","func":"Transfer","args":["g1kcdd3n0d472g2p5l8svyg9t0wq6h5857nq992f","1"]}],"fee":{"gas_wanted":"9000000","gas_fee":"1ugnot"},"signatures":[{"pub_key":{"@type":"/tm.PubKeySecp256k1","value":"A+FhNtsXHjLfSJk1lB8FbiL4mGPjc50Kt81J7EKDnJ2y"},"signature":"6Jk3gs564wGTutNdODztNUlg88/WHmMPmJGRZHDuV00Mc9M5gGWBZDEpGysLsqzjMDxmsTu1PLtTYfTj0KphGQ=="}],"memo":""}';
    const signedTx = strToSignedTx(str)!;
    const document = txToDocument(signedTx);

    expect(document.msgs).toHaveLength(1);
    expect(document.msgs[0].type).toBe('/vm.m_call');
    expect(document.fee.gas).toBe('9000000');
    expect(signedTx.signatures).toHaveLength(1);
  });

  it('str to base64', async () => {
    const str =
      '{"msg":[{"@type":"/vm.m_call","caller":"g1jg8mtutu9khhfwc4nxmuhcpftf0pajdhfvsqf5","send":"","pkg_path":"gno.land/r/demo/tong","func":"Transfer","args":["g1kcdd3n0d472g2p5l8svyg9t0wq6h5857nq992f","1"]}],"fee":{"gas_wanted":"9000000","gas_fee":"1ugnot"},"signatures":[{"pub_key":{"@type":"/tm.PubKeySecp256k1","value":"A+FhNtsXHjLfSJk1lB8FbiL4mGPjc50Kt81J7EKDnJ2y"},"signature":"6Jk3gs564wGTutNdODztNUlg88/WHmMPmJGRZHDuV00Mc9M5gGWBZDEpGysLsqzjMDxmsTu1PLtTYfTj0KphGQ=="}],"memo":""}';
    const signedTx = strToSignedTx(str)!;
    const document = txToDocument(signedTx);

    expect(document.msgs[0].value.func).toBe('Transfer');
    expect(document.msgs[0].value.args).toEqual([
      'g1kcdd3n0d472g2p5l8svyg9t0wq6h5857nq992f',
      '1',
    ]);
  });
});
