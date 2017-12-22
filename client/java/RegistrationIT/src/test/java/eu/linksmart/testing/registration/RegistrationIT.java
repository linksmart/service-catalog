package eu.linksmart.testing.registration;

import io.swagger.client.ApiClient;
import io.swagger.client.api.ScApi;
import io.swagger.client.model.APIIndex;
import io.swagger.client.model.Service;
import io.swagger.client.model.ServiceDocs;
import org.junit.Test;

import java.math.BigDecimal;
import java.util.HashMap;
import java.util.Map;
import java.util.UUID;

import static org.junit.Assert.assertArrayEquals;
import static org.junit.Assert.assertTrue;
import static org.junit.Assert.fail;

/**
 * Created by José Ángel Carvajal on 21.12.2017 a researcher of Fraunhofer FIT.
 */
public class RegistrationIT {

    static final String BASE_URL_PATH = "base_url", BASE_URL = "http://localhost:8082";
    @Test
    public void registration(){
        if(!System.getenv().containsKey("integration_test"))
            return;
        ApiClient client = new ApiClient();

        client.setBasePath((System.getenv().containsKey(BASE_URL_PATH))?System.getenv().get(BASE_URL_PATH):BASE_URL);

        ScApi api = new ScApi(client);

        String id = UUID.randomUUID().toString();

        Service service = new Service();
        service.setName("_it._tcp");
        Map<String,String> apis = new HashMap<String, String>();
        apis.put("Test API","http://test:666");
        // service.meta("test");
        service.apis(apis);
        ServiceDocs docs = new ServiceDocs();
        docs.addApisItem("Test API");
        docs.description("it's a test!");
        docs.setType("application/json");
        docs.setUrl("http://test:666/docu");
        service.addDocsItem(docs);

        try{
            Service service1;

            api.idPut(id,service);

            service1 = api.idGet(id);

            assertTrue("Ids most be equal", id.equals(service1.getId()));
            APIIndex index =  api.rootGet(new BigDecimal(1),new BigDecimal(100));
            assertTrue("It must contain 1 service", index.getTotal().equals(1));
            api.idDelete(id);

        }catch (Exception e){
            e.printStackTrace();
            fail();
        }



    }
}
